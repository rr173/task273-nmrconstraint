package service

import (
	"context"

	"task273-nmrconstraint/internal/diagnose"
	"task273-nmrconstraint/internal/model"
	"task273-nmrconstraint/internal/propagate"
	"task273-nmrconstraint/internal/store"
)

// SolveResult 是一次求解的输出摘要。
type SolveResult struct {
	BatchID       int64                  `json:"batch_id"`
	Iterations    int                    `json:"iterations"`
	Converged     bool                   `json:"converged"`
	HasConflict   bool                   `json:"has_conflict"`
	ConflictKind  *model.ConflictKind    `json:"conflict_kind,omitempty"`
	Violations    []diagnose.Violation   `json:"violations"`
	InvertedEdges int                    `json:"inverted_edges"`
	EdgeCount     int                    `json:"edge_count"`
	BatchStatus   model.BatchStatus      `json:"batch_status"`
}

// DiagnosisService 编排求解：传播 → 冲突检测 → 冲突集 → 最小化 → 豁免。
type DiagnosisService struct {
	batches *store.BatchStore
	cons    *store.ConstraintStore
	bounds  *store.BoundStore
	sets    *store.ConflictSetStore
	exemps  *store.ExemptionStore
}

// NewDiagnosisService 构造诊断服务。
func NewDiagnosisService(batches *store.BatchStore, cons *store.ConstraintStore, bounds *store.BoundStore, sets *store.ConflictSetStore, exemps *store.ExemptionStore) *DiagnosisService {
	return &DiagnosisService{
		batches: batches,
		cons:    cons,
		bounds:  bounds,
		sets:    sets,
		exemps:  exemps,
	}
}

// Solve 对批次执行一次完整求解并落盘传播边界。
// ctx 取消时不得落盘半成品边界或批次状态。
func (s *DiagnosisService) Solve(ctx context.Context, batchID int64) (*SolveResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	b, err := s.batches.Get(batchID)
	if err != nil {
		return nil, err
	}
	if b.Status == model.BatchSealed || b.Status == model.BatchPublished {
		return nil, model.ErrNotSolvable
	}
	constraints, err := s.cons.List(batchID)
	if err != nil {
		return nil, err
	}
	active := make([]*model.Constraint, 0, len(constraints))
	for _, c := range constraints {
		if c.Active() {
			active = append(active, c)
		}
	}
	if len(active) == 0 {
		return nil, model.ErrEmptyConstraintSet
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	res := propagate.Propagate(active)
	violations := diagnose.DetectViolations(res.Edges)
	hasConflict := len(res.Inverted) > 0 || len(violations) > 0

	// 更新约束状态。
	involved := make(map[int64]bool)
	for _, e := range res.Inverted {
		involved[e.ConstraintID] = true
	}
	for _, v := range violations {
		involved[v.ConstraintID] = true
	}
	for _, c := range active {
		status := model.ConstraintValid
		if involved[c.ID] {
			status = model.ConstraintConflicted
		}
		if c.Status != status {
			if err := s.cons.UpdateStatus(c.ID, status); err != nil {
				return nil, err
			}
		}
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// 落盘传播边界（幂等替换）。
	if err := s.bounds.ReplaceBatch(ctx, batchID, res.Edges); err != nil {
		return nil, err
	}

	// 更新批次状态：receiving 先进入 solving，再按冲突结果流转。
	if b.Status == model.BatchReceiving {
		if err := b.Advance(model.BatchSolving); err != nil {
			return nil, err
		}
		if err := s.batches.UpdateStatus(batchID, b.Status, b.SealedAt); err != nil {
			return nil, err
		}
	}
	target := model.BatchReleasable
	if hasConflict {
		target = model.BatchConflicted
	}
	if b.Status != target {
		if b.Status == model.BatchReleasable && target == model.BatchConflicted {
			if err := b.Advance(model.BatchSolving); err != nil {
				return nil, err
			}
			if err := s.batches.UpdateStatus(batchID, b.Status, b.SealedAt); err != nil {
				return nil, err
			}
		}
		if err := b.Advance(target); err != nil {
			return nil, err
		}
		if err := s.batches.UpdateStatus(batchID, b.Status, b.SealedAt); err != nil {
			return nil, err
		}
	}

	var kind *model.ConflictKind
	if hasConflict {
		k := diagnose.ConflictKindOf(res.Inverted, violations)
		kind = &k
	}
	return &SolveResult{
		BatchID:       batchID,
		Iterations:    res.Iterations,
		Converged:     res.Converged,
		HasConflict:   hasConflict,
		ConflictKind:  kind,
		Violations:    violations,
		InvertedEdges: len(res.Inverted),
		EdgeCount:     len(res.Edges),
		BatchStatus:   target,
	}, nil
}

// BuildConflictSets 为批次生成候选冲突集（每条冲突约束集合一个）。
func (s *DiagnosisService) BuildConflictSets(batchID int64) ([]*model.ConflictSet, error) {
	b, err := s.batches.Get(batchID)
	if err != nil {
		return nil, err
	}
	if b.Status != model.BatchConflicted {
		return nil, model.ErrConflictNotPresent
	}
	constraints, err := s.cons.List(batchID)
	if err != nil {
		return nil, err
	}
	var conflicted []*model.Constraint
	for _, c := range constraints {
		if c.Status == model.ConstraintConflicted {
			conflicted = append(conflicted, c)
		}
	}
	if len(conflicted) == 0 {
		return nil, model.ErrConflictNotPresent
	}

	// 判定冲突类型：以当前传播结果为准。
	edges, err := s.bounds.List(batchID)
	if err != nil {
		return nil, err
	}
	violations := diagnose.DetectViolations(edges)
	kind := model.ConflictTriangle
	invertedPresent := false
	for _, e := range edges {
		if e.Inverted() {
			invertedPresent = true
			break
		}
	}
	if invertedPresent {
		kind = model.ConflictInterval
	} else if len(violations) == 0 {
		kind = model.ConflictInterval
	}

	cs := model.NewConflictSet(batchID, kind)
	members := make([]*model.ConflictMember, 0, len(conflicted))
	shared := model.NewConflictMember(0, 0)
	for _, c := range conflicted {
		shared.ConstraintID = c.ID
		members = append(members, shared)
	}
	return []*model.ConflictSet{cs}, s.createSet(cs, members)
}

func (s *DiagnosisService) createSet(cs *model.ConflictSet, members []*model.ConflictMember) error {
	_, err := s.sets.Create(cs, members)
	return err
}

// Minimize 对冲突集执行最小化（MUC 求解）。
func (s *DiagnosisService) Minimize(setID int64) (*model.ConflictSet, []*model.ConflictMember, error) {
	cs, err := s.sets.Get(setID)
	if err != nil {
		return nil, nil, err
	}
	if cs.Minimized {
		return nil, nil, model.ErrMinimizeDone
	}
	members, err := s.sets.Members(setID)
	if err != nil {
		return nil, nil, err
	}
	ids := make([]int64, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.ConstraintID)
	}
	constraints, err := s.cons.ByIDs(cs.BatchID, ids)
	if err != nil {
		return nil, nil, err
	}
	minimal, ok := diagnose.MinimalConflictSet(constraints)
	if !ok {
		// 理论上不会发生：候选集合本身就是冲突的。
		return nil, nil, model.ErrConflictNotPresent
	}
	necessary := make(map[int64]bool, len(minimal))
	for _, c := range minimal {
		necessary[c.ID] = true
	}
	for _, m := range members {
		removed := !necessary[m.ConstraintID]
		if m.Removed != removed {
			if err := s.sets.UpdateMemberRemoved(setID, m.ConstraintID, removed); err != nil {
				return nil, nil, err
			}
		}
		m.Removed = removed
	}
	cs.MarkMinimized()
	cs.MemberCount = len(minimal)
	if err := s.sets.Update(cs); err != nil {
		return nil, nil, err
	}
	return cs, members, nil
}

// Exempt 豁免一条冲突约束：记录豁免、排除约束、豁免相关冲突集并重新评估批次。
func (s *DiagnosisService) Exempt(batchID, constraintID int64, reason string) (*model.Exemption, error) {
	b, err := s.batches.Get(batchID)
	if err != nil {
		return nil, err
	}
	if b.Status == model.BatchSealed {
		return nil, model.ErrSealedBatchImmutable
	}
	c, err := s.cons.GetInBatch(batchID, constraintID)
	if err != nil {
		return nil, err
	}
	c.Exclude()
	if err := s.cons.UpdateStatus(constraintID, c.Status); err != nil {
		return nil, err
	}
	e := model.NewExemption(batchID, constraintID, reason)
	created, err := s.exemps.Create(e)
	if err != nil {
		return nil, err
	}
	// 豁免包含该约束的全部激活冲突集。
	if err := s.exemptSetsWithConstraint(batchID, constraintID); err != nil {
		return nil, err
	}
	// 若不再有激活的冲突集，批次推进为可发布。
	active, err := s.sets.HasActiveConflicts(batchID)
	if err != nil {
		return nil, err
	}
	if !active && (b.Status == model.BatchConflicted || b.Status == model.BatchSolving) {
		if err := b.Advance(model.BatchReleasable); err != nil {
			return nil, err
		}
		if err := s.batches.UpdateStatus(batchID, b.Status, b.SealedAt); err != nil {
			return nil, err
		}
	}
	return created, nil
}

// exemptSetsWithConstraint 将包含指定约束成员的激活冲突集标记为豁免。
func (s *DiagnosisService) exemptSetsWithConstraint(batchID, constraintID int64) error {
	sets, err := s.sets.List(batchID)
	if err != nil {
		return err
	}
	for _, cs := range sets {
		if cs.Status != model.ConflictSetCandidate && cs.Status != model.ConflictSetConfirmed {
			continue
		}
		members, err := s.sets.Members(cs.ID)
		if err != nil {
			return err
		}
		contains := false
		for _, m := range members {
			if m.ConstraintID == constraintID {
				contains = true
				break
			}
		}
		if contains {
			cs.Exempt()
			if err := s.sets.Update(cs); err != nil {
				return err
			}
		}
	}
	return nil
}

// Exemptions 列出批次全部豁免。
func (s *DiagnosisService) Exemptions(batchID int64) ([]*model.Exemption, error) {
	return s.exemps.List(batchID)
}

// ListConflictSets 列出批次全部冲突集。
func (s *DiagnosisService) ListConflictSets(batchID int64) ([]*model.ConflictSet, error) {
	return s.sets.List(batchID)
}

// GetConflictSet 读取冲突集详情。
func (s *DiagnosisService) GetConflictSet(setID int64) (*model.ConflictSet, []*model.ConflictMember, error) {
	cs, err := s.sets.Get(setID)
	if err != nil {
		return nil, nil, err
	}
	members, err := s.sets.Members(setID)
	if err != nil {
		return nil, nil, err
	}
	return cs, members, nil
}

// ListBounds 列出批次传播边界。
func (s *DiagnosisService) ListBounds(batchID int64) ([]*model.BoundEdge, error) {
	return s.bounds.List(batchID)
}

// ListViolations 重新检测批次三角不等式违反。
func (s *DiagnosisService) ListViolations(batchID int64) ([]diagnose.Violation, error) {
	edges, err := s.bounds.List(batchID)
	if err != nil {
		return nil, err
	}
	return diagnose.DetectViolations(edges), nil
}

// ListConflicted 列出批次冲突约束。
func (s *DiagnosisService) ListConflicted(batchID int64) ([]*model.Constraint, error) {
	constraints, err := s.cons.List(batchID)
	if err != nil {
		return nil, err
	}
	var out []*model.Constraint
	for _, c := range constraints {
		if c.Status == model.ConstraintConflicted {
			out = append(out, c)
		}
	}
	return out, nil
}

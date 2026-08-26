// task273-nmrconstraint 蛋白质 NMR 距离约束冲突诊断服务入口。
//
// 用法：
//
//	task273-nmrconstraint --addr :8080 --db task273-nmrconstraint.db   # 启动 HTTP 服务
//	task273-nmrconstraint --smoke-test [--db <临时库>]                   # 自检：真实闭环 + 重启恢复验证
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"task273-nmrconstraint/internal/constraint"
	"task273-nmrconstraint/internal/httpapi"
	"task273-nmrconstraint/internal/mapping"
	"task273-nmrconstraint/internal/model"
	"task273-nmrconstraint/internal/service"
	"task273-nmrconstraint/internal/snapshot"
	"task273-nmrconstraint/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "task273-nmrconstraint.db", "SQLite database path")
	smoke := flag.Bool("smoke-test", false, "run end-to-end smoke test and exit")
	flag.Parse()

	if *smoke {
		if err := runSmokeTest(*dbPath); err != nil {
			log.Fatalf("smoke test failed: %v", err)
		}
		fmt.Println("smoke test passed")
		return
	}

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	handler := buildHandler(db)
	log.Printf("task273-nmrconstraint listening on %s (db=%s)", *addr, *dbPath)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// buildHandler 组装依赖并返回 HTTP handler。
func buildHandler(db *store.DB) http.Handler {
	batchStore := store.NewBatchStore(db)
	atomStore := store.NewAtomStore(db)
	peakStore := store.NewPeakStore(db)
	constraintStore := store.NewConstraintStore(db)
	boundStore := store.NewBoundStore(db)
	setStore := store.NewConflictSetStore(db)
	exemptionStore := store.NewExemptionStore(db)
	snapshotStore := store.NewSnapshotStore(db)

	batches := service.NewBatchService(batchStore)
	atoms := mapping.NewService(atomStore, batchStore)
	peaks := service.NewPeakService(peakStore, batchStore)
	cons := constraint.NewService(batchStore, atomStore, peakStore, constraintStore)
	diag := service.NewDiagnosisService(batchStore, constraintStore, boundStore, setStore, exemptionStore)
	snaps := snapshot.NewService(batchStore, constraintStore, boundStore, snapshotStore, exemptionStore, peaks.ConfidenceVersion)

	return httpapi.NewServer(db, batches, atoms, peaks, cons, diag, snaps).Handler()
}

// runSmokeTest 执行端到端自检：
// 创建批次 → 导入原子/峰/约束 → 求解发现三角不等式冲突 → 冲突集最小化
// → 豁免冲突约束 → 重新求解可发布 → 发布快照 → 关闭重开数据库验证恢复。
func runSmokeTest(dbPath string) error {
	if dbPath == "" || dbPath == "task273-nmrconstraint.db" {
		dbPath = "task273-nmrconstraint-smoke.db"
	}
	_ = os.Remove(dbPath)

	db, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	batchStore := store.NewBatchStore(db)
	atomStore := store.NewAtomStore(db)
	peakStore := store.NewPeakStore(db)
	constraintStore := store.NewConstraintStore(db)
	boundStore := store.NewBoundStore(db)
	setStore := store.NewConflictSetStore(db)
	exemptionStore := store.NewExemptionStore(db)
	snapshotStore := store.NewSnapshotStore(db)

	batches := service.NewBatchService(batchStore)
	atoms := mapping.NewService(atomStore, batchStore)
	peaks := service.NewPeakService(peakStore, batchStore)
	cons := constraint.NewService(batchStore, atomStore, peakStore, constraintStore)
	diag := service.NewDiagnosisService(batchStore, constraintStore, boundStore, setStore, exemptionStore)
	snaps := snapshot.NewService(batchStore, constraintStore, boundStore, snapshotStore, exemptionStore, peaks.ConfidenceVersion)

	// 1. 创建批次。
	batch, err := batches.Create("NMR-Triangle-Demo", "三个距离区间违反三角不等式的最小冲突诊断演示")
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	fmt.Printf("[1] batch created id=%d status=%s\n", batch.ID, batch.Status)

	// 2. 导入原子。
	atomsImported, err := atoms.Import(batch.ID, []*model.Atom{
		model.NewAtom(0, "A", "GLY", "C"),
		model.NewAtom(0, "B", "GLY", "N"),
		model.NewAtom(0, "C", "ALA", "C"),
		model.NewAtom(0, "D", "ALA", "N"),
	})
	if err != nil {
		return fmt.Errorf("import atoms: %w", err)
	}
	byName := map[string]*model.Atom{}
	for _, a := range atomsImported {
		byName[a.Name] = a
	}
	fmt.Printf("[2] atoms imported count=%d\n", len(atomsImported))

	// 3. 导入 NOE 峰。
	peaksImported, err := peaks.Import(batch.ID, []*model.NoePeak{
		model.NewNoePeak(0, "pAB", byName["A"].ID, byName["B"].ID, 5.0, 0.95),
		model.NewNoePeak(0, "pAC", byName["A"].ID, byName["C"].ID, 4.0, 0.90),
		model.NewNoePeak(0, "pBC", byName["B"].ID, byName["C"].ID, 0.2, 0.85),
	})
	if err != nil {
		return fmt.Errorf("import peaks: %w", err)
	}
	peakByName := map[string]*model.NoePeak{}
	for _, p := range peaksImported {
		peakByName[p.Name] = p
	}
	fmt.Printf("[3] peaks imported count=%d\n", len(peaksImported))

	// 4. 创建距离约束：BC 的 [12,15] 违反三角不等式
	//    （BC 下界 12 > AB 上界 6 + AC 上界 5 = 11）。
	constraints, err := cons.CreateFromPeaks(batch.ID, []*model.Constraint{
		model.NewConstraint(0, peakByName["pAB"].ID, byName["A"].ID, byName["B"].ID, 4.0, 6.0),
		model.NewConstraint(0, peakByName["pAC"].ID, byName["A"].ID, byName["C"].ID, 3.0, 5.0),
		model.NewConstraint(0, peakByName["pBC"].ID, byName["B"].ID, byName["C"].ID, 12.0, 15.0),
	})
	if err != nil {
		return fmt.Errorf("create constraints: %w", err)
	}
	fmt.Printf("[4] constraints created count=%d\n", len(constraints))

	// 5. 求解：预期发现冲突。
	result, err := diag.Solve(context.Background(), batch.ID)
	if err != nil {
		return fmt.Errorf("solve: %w", err)
	}
	if !result.HasConflict {
		return fmt.Errorf("expected conflict, got none (iterations=%d)", result.Iterations)
	}
	fmt.Printf("[5] solve detected conflict kind=%v violations=%d inverted=%d iterations=%d status=%s\n",
		*result.ConflictKind, len(result.Violations), result.InvertedEdges, result.Iterations, result.BatchStatus)

	// 6. 构建候选冲突集。
	sets, err := diag.BuildConflictSets(batch.ID)
	if err != nil {
		return fmt.Errorf("build conflict sets: %w", err)
	}
	fmt.Printf("[6] conflict sets built count=%d kind=%s members=%d\n", len(sets), sets[0].Kind, sets[0].MemberCount)

	// 7. 最小化冲突集。
	setID := sets[0].ID
	cs, members, err := diag.Minimize(setID)
	if err != nil {
		return fmt.Errorf("minimize: %w", err)
	}
	fmt.Printf("[7] conflict set minimized id=%d minimized=%v members=%d\n", cs.ID, cs.Minimized, cs.MemberCount)
	for _, m := range members {
		if !m.Removed {
			fmt.Printf("    necessary constraint id=%d\n", m.ConstraintID)
		}
	}

	// 8. 豁免最小冲突集中的一条必要约束（三角违反中任取一条即可解除），
	//    以峰重叠为由。其余约束保持有效，重新求解应干净。
	exempted := int64(0)
	exemptReason := "peak overlap suspected: BC cross-peak contaminated"
	for _, m := range members {
		if m.Removed {
			continue
		}
		ex, err := diag.Exempt(batch.ID, m.ConstraintID, exemptReason)
		if err != nil {
			return fmt.Errorf("exempt: %w", err)
		}
		exempted = ex.ConstraintID
		fmt.Printf("[8] exempted constraint id=%d reason=%q\n", ex.ConstraintID, ex.Reason)
		break
	}
	if exempted == 0 {
		return fmt.Errorf("expected at least one exemption")
	}

	// 9. 重新求解：预期可发布。
	result2, err := diag.Solve(context.Background(), batch.ID)
	if err != nil {
		return fmt.Errorf("re-solve: %w", err)
	}
	if result2.HasConflict {
		return fmt.Errorf("expected no conflict after exemption, still has %v", *result2.ConflictKind)
	}
	fmt.Printf("[9] re-solve clean status=%s edges=%d\n", result2.BatchStatus, result2.EdgeCount)

	// 10. 创建并发布诊断快照。
	snap, err := snaps.CreateDraft(batch.ID, "diagnosis-v1")
	if err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}
	published, err := snaps.Publish(batch.ID, snap.ID)
	if err != nil {
		return fmt.Errorf("publish snapshot: %w", err)
	}
	fmt.Printf("[10] snapshot published id=%d status=%s version=%d\n", published.ID, published.Status, published.ConfidenceVersion)

	// 11. 关闭并重新打开数据库，验证持久化与恢复。
	batchID := batch.ID
	snapshotID := snap.ID
	exemptionConstraintID := exempted
	if err := db.Close(); err != nil {
		return fmt.Errorf("close db: %w", err)
	}
	db2, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("reopen db: %w", err)
	}
	defer func() { _ = db2.Close() }()

	batchStore2 := store.NewBatchStore(db2)
	recoveredBatch, err := batchStore2.Get(batchID)
	if err != nil {
		return fmt.Errorf("recover batch: %w", err)
	}
	if recoveredBatch.Status != model.BatchPublished {
		return fmt.Errorf("recovered batch status=%s, want published", recoveredBatch.Status)
	}
	snapshotStore2 := store.NewSnapshotStore(db2)
	recoveredSnap, err := snapshotStore2.Get(snapshotID)
	if err != nil {
		return fmt.Errorf("recover snapshot: %w", err)
	}
	if recoveredSnap.Status != model.SnapshotPublished {
		return fmt.Errorf("recovered snapshot status=%s, want published", recoveredSnap.Status)
	}
	items, err := snapshotStore2.Items(snapshotID)
	if err != nil {
		return fmt.Errorf("recover snapshot items: %w", err)
	}
	exemptionStore2 := store.NewExemptionStore(db2)
	exemptions, err := exemptionStore2.List(batchID)
	if err != nil {
		return fmt.Errorf("recover exemptions: %w", err)
	}
	if len(exemptions) == 0 || exemptions[0].ConstraintID != exemptionConstraintID {
		return fmt.Errorf("recovered exemptions mismatch")
	}
	fmt.Printf("[11] restart recovery ok batch=%s snapshot=%s items=%d exemptions=%d\n",
		recoveredBatch.Status, recoveredSnap.Status, len(items), len(exemptions))

	return nil
}

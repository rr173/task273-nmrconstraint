package model

// BatchStatus 描述结构批次从接收约束到封存诊断的完整生命周期。
type BatchStatus string

const (
	BatchReceiving  BatchStatus = "receiving"  // 接收中：原子/峰/约束可导入
	BatchSolving    BatchStatus = "solving"    // 求解中：距离界传播与冲突检测已触发
	BatchConflicted BatchStatus = "conflicted" // 存在冲突：检测到三角不等式违反或区间倒置
	BatchReleasable BatchStatus = "releasable" // 可发布：无未豁免冲突
	BatchPublished  BatchStatus = "published"  // 已发布：诊断快照已发布
	BatchSealed     BatchStatus = "sealed"     // 封存：批次不可再修改
)

// ConstraintStatus 描述单条距离约束的归属与一致性状态。
type ConstraintStatus string

const (
	ConstraintRaw       ConstraintStatus = "raw"       // 原始：由 NOE 峰归属生成
	ConstraintValid     ConstraintStatus = "valid"     // 有效：参与传播且无冲突
	ConstraintConflicted ConstraintStatus = "conflicted" // 冲突：导致三角不等式违反或区间倒置
	ConstraintExcluded  ConstraintStatus = "excluded"  // 排除：研究者标记不可信
)

// PeakStatus 描述 NOE 峰的观察状态。
type PeakStatus string

const (
	PeakRaw     PeakStatus = "raw"     // 原始
	PeakValid   PeakStatus = "valid"   // 有效
	PeakOverlap PeakStatus = "overlap" // 重叠可疑：可能混入其它峰信号
	PeakExcluded PeakStatus = "excluded" // 排除
)

// AtomStatus 描述原子映射的可用状态。
type AtomStatus string

const (
	AtomRaw     AtomStatus = "raw"
	AtomActive  AtomStatus = "active"
	AtomExcluded AtomStatus = "excluded"
)

// ConflictKind 描述冲突集的类型。
type ConflictKind string

const (
	ConflictTriangle ConflictKind = "triangle" // 三角不等式违反
	ConflictInterval ConflictKind = "interval" // 区间倒置（传播后 lo > hi）
)

// ConflictSetStatus 描述冲突集的诊断状态。
type ConflictSetStatus string

const (
	ConflictSetCandidate ConflictSetStatus = "candidate" // 候选：由检测器生成
	ConflictSetMinimizing ConflictSetStatus = "minimizing" // 最小化中
	ConflictSetConfirmed ConflictSetStatus = "confirmed" // 确认：研究者接受该冲突集
	ConflictSetExempted  ConflictSetStatus = "exempted"  // 豁免：研究者豁免其成员
)

// SnapshotStatus 描述诊断快照的状态。
type SnapshotStatus string

const (
	SnapshotDraft      SnapshotStatus = "draft"
	SnapshotPublished  SnapshotStatus = "published"
	SnapshotSuperseded SnapshotStatus = "superseded"
)

// ValidTransitions 定义批次状态机的合法流转边。
var ValidTransitions = map[BatchStatus][]BatchStatus{
	BatchReceiving:  {BatchSolving, BatchSealed},
	BatchSolving:    {BatchConflicted, BatchReleasable, BatchSealed},
	BatchConflicted: {BatchSolving, BatchReleasable, BatchSealed},
	BatchReleasable: {BatchPublished, BatchSolving, BatchSealed},
	BatchPublished:  {BatchSealed},
	BatchSealed:     {},
}

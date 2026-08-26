# BENZHI 评测说明

基于 Go 实现的蛋白质 NMR 距离约束冲突诊断后端服务，一款后端服务，完成原子映射与 NOE 峰导入、三角不等式距离界传播、最小冲突集求解与不可变诊断快照发布。

## 启动

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go run ./cmd/task273-nmrconstraint --addr :8080 --db task273-nmrconstraint.db
```

## 自检（不启动长驻服务）

```bash
go run ./cmd/task273-nmrconstraint --smoke-test
```

`--smoke-test` 会真实创建结构批次、导入原子/峰/距离约束、传播距离界、最小化冲突集、豁免后重新求解并发布诊断快照，关闭并重开同一数据库验证持久化，最后以 0 退出码结束。

## 构建门禁

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/task273-nmrconstraint --smoke-test
```

## HTTP API（前缀 /api）

- 批次：POST/GET /api/batches、GET/PATCH /api/batches/{id}、POST /api/batches/{id}/advance
- 原子映射：POST/GET /api/batches/{id}/atoms、GET /api/atoms/{id}、POST /api/atoms/{id}/exclude、POST /api/atoms/{id}/activate
- NOE 峰：POST/GET /api/batches/{id}/peaks、PATCH /api/peaks/{id}、POST /api/peaks/{id}/overlap、POST /api/peaks/{id}/exclude
- 距离约束：POST/GET /api/batches/{id}/constraints、GET /api/constraints/{id}、POST /api/constraints/{id}/exclude、POST /api/constraints/{id}/restore
- 求解与冲突：POST /api/batches/{id}/solve、GET /api/batches/{id}/bounds、GET /api/batches/{id}/violations、GET /api/batches/{id}/conflicts
- 冲突集与豁免：POST/GET /api/batches/{id}/conflictsets、GET /api/conflictsets/{id}、POST /api/conflictsets/{id}/minimize、POST/GET /api/batches/{id}/exemptions
- 快照：POST/GET /api/batches/{id}/snapshots、GET /api/snapshots/{id}、POST /api/snapshots/{id}/publish
- 其他：GET /api/health

## 持久化

SQLite（modernc.org/sqlite，CGO 无关）。十表：batches、atoms、noe_peaks、constraints、bound_edges、conflict_sets、conflict_members、exemptions、snapshots、snapshot_items。原子名/峰名/原子对约束/豁免唯一；已发布快照条目与 live 传播边界解耦，重启同一数据库可恢复批次、快照与豁免。

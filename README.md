# task273-nmrconstraint 蛋白质 NMR 距离约束冲突诊断服务

## 业务背景

结构生物学研究者通过 NOE（核欧氏效应）交叉峰获得原子间距离约束，用于求解蛋白质三维构象。当一组距离区间无法被同一构象同时满足时——典型表现为三角不等式违反（如 `lo_AB > hi_AC + hi_CB`）或距离界传播后区间倒置——构象优化持续无法收敛。本服务持久化原子映射、NOE 峰与距离约束，构造约束图并以三角不等式传播收紧距离界，定位最小冲突集（MUC），支持峰重叠豁免与重新求解，最终发布不可变诊断快照。

## 快速开始

```bash
# 端到端自检（含重启恢复验证，退出码 0 即通过）
go run ./cmd/task273-nmrconstraint --smoke-test

# 启动服务
go run ./cmd/task273-nmrconstraint --addr :8080 --db task273-nmrconstraint.db
```

## 典型使用流程

1. `POST /api/batches` 创建批次；
2. `POST /api/batches/{id}/atoms` 导入原子映射；
3. `POST /api/batches/{id}/peaks` 导入 NOE 峰（含强度与观察置信度）；
4. `POST /api/batches/{id}/constraints` 依据峰归属距离区间；
5. `POST /api/batches/{id}/solve` 传播距离界并检测冲突；
6. `GET /api/batches/{id}/violations` 查看三角不等式违反证据；
7. `POST /api/batches/{id}/conflictsets` 生成候选冲突集，`POST /api/conflictsets/{id}/minimize` 求 MUC；
8. `POST /api/batches/{id}/exemptions` 豁免不可信约束（如峰重叠）；
9. 重新 `solve` 至可发布后，`POST /api/batches/{id}/snapshots` + `POST /api/snapshots/{id}/publish` 发布诊断快照。

## 目录结构

```
├── go.mod / go.sum
├── component-versions.json        # Go 与 SQLite 版本锁
├── Dockerfile / benzhi.Dockerfile
├── build_benzhi_docker.sh
├── BENZHI_README.md / README.md
├── cmd/task273-nmrconstraint/     # 入口（--addr / --db / --smoke-test）
└── internal/
    ├── model/                     # 实体、枚举、领域错误
    ├── store/                     # SQLite 迁移与全部仓储
    ├── mapping/                   # 原子映射领域
    ├── constraint/                # 峰→约束归属与校验
    ├── propagate/                 # 三角不等式距离界传播
    ├── diagnose/                  # 冲突检测 + MUC 最小化
    ├── snapshot/                  # 诊断快照发布
    ├── service/                   # 用例编排（批次/峰/诊断）
    └── httpapi/                   # HTTP 层（/api 前缀）
```

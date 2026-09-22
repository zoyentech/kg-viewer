# 3D 知识图谱（Neo4j + `/kg-viewer`）

> `GET /v1/graph/kg` 把 Neo4j 里的图谱快照读出来，喂给内嵌的 Three.js 3D
> 可视化页面 `/kg-viewer`。两个端点同源，浏览器直接打开页面就能看到真实图谱。

## 架构

```
┌─────────────────── 浏览器 ─────────────────────┐
│ /kg-viewer  (Three.js · 内嵌 HTML)               │
│   └─ fetch('${origin}/v1/graph/kg')              │  ← 不传 limit, 全量
└────────────────────┬───────────────────────────┘
                     │ JSON: { nodes[], links[], source }
┌────────────────────▼───────────────────────────┐
│ Go 后端  /v1/graph/kg (handler.GraphHandler)    │
│   └─ neo4j.Client.Fetch(limit, label)            │
│      └─ profile-selected Bolt endpoint           │
└────────────────────┬───────────────────────────┘
                     │ Cypher
┌────────────────────▼───────────────────────────┐
│ Neo4j 5.26  (docker compose @ 127.0.0.1:7687)   │
│   seed: deploy/neo4j/seeds/seed.cypher          │
└─────────────────────────────────────────────────┘
```

- **不在 API 契约里**：`/kg-viewer` 是静态 HTML（`internal/viewer/index.html`），
  通过 `//go:embed` 打到 `bin/api` 里；和 `/health` 一样在 `pkg/httpx/coverage_test.go`
  的 `routerOnlyOps` 中显式登记，不算 OpenAPI operation。
- **公开端点**：`/v1/graph/kg` 是 OpenAPI `Graph` 标签下的唯一 operation，
  鉴权关闭 (`security: []`)，调用方只有浏览器 + 调试用 curl。
- **图谱查询**：UNION 三分支——直连边 + Ingredient-(EC)-Evidence 穿透 +
  Evidence-(EC)-HealthTopic 穿透。EvidenceContext 节点不返回，仅作桥。
  `limit` > 0 时每个分支独立采样(`LIMIT n`)，不传则全量返回。
- **故障态**：未配 Neo4j → 503；查询失败 → 502；客户端不会因为后端挂掉而
  拿到 500。3D 页面在 fetch 失败时自动回退到内置 `SAMPLE` 数据。

## 一键起库灌数

```bash
make neo4j-up        # docker compose 起 Neo4j 5.26 Community（7474/7687）
make neo4j-seed      # 灌 deploy/neo4j/seeds/seed.cypher（幂等 MERGE）
make neo4j-logs      # 跟日志
make neo4j-down      # 停容器
```

浏览器管理台：<http://localhost:7474>（`neo4j` / `neo4j-healthdinner-2026`）。

## 后端 env

使用 `KG_GRAPH_PROFILE` 选择图谱版本和显示语言。旧版英文/中文分别连接
M3 的 `7687` / `7688`；v2 的 `v2-en` / `v2-zh` 共用 `7689`，只切换
`name_en` / `name_zh` 等双语属性。

```bash
KG_GRAPH_PROFILE=v2-zh
KG_GRAPH_V2_URI=bolt://192.168.31.201:7689
KG_GRAPH_V2_DATABASE=neo4j
KG_GRAPH_LEGACY_EN_URI=bolt://192.168.31.201:7687
KG_GRAPH_LEGACY_ZH_URI=bolt://192.168.31.201:7688
```

`NEO4J_URI` 留空 → `/v1/graph/kg` 始终 503，前端 3D 页面
自动用 `SAMPLE` 兜底。`make backend-target status` / `make server-dev`
都会沿用这些 env。

## API

### `GET /v1/graph/kg`

| 查询参数 | 类型 | 默认 | 说明 |
| --- | --- | --- | --- |
| `limit` | int (>=1) | 0（=全量） | 单关系类型采样上限；不传返回所有关系 |
| `label` | string | "" | 按节点标签过滤（如 `Food` / `Nutrient`），空=全图 |

**固定 4 类节点**：`Product` / `Ingredient` / `Evidence` / `HealthTopic`。
`EvidenceContext` 作为隐式桥(不出现在 nodes 中)，通过两跳穿透产生
`EVIDENCE_FOR` (Ingredient→Evidence) 和 `SUPPORTS_TOPIC` (Evidence→HealthTopic)
两种虚拟边，使 Evidence 视觉上居于 Ingredient 与 HealthTopic 之间：

```
Product → Ingredient --(EC)-- Evidence --(EC)-- HealthTopic
```

直接边(`DECLARES_INGREDIENT` 等)照常返回。

返回壳：

```json
{
  "code": 0,
  "data": {
    "nodes": [
      { "id": "4:...:0", "label": "营养与健康", "type": "Domain" }
    ],
    "links": [
      { "source": "4:...:0", "target": "4:...:1", "type": "CONTAINS" }
    ],
    "source": "neo4j"
  }
}
```

错误码：

- `502 Bad Gateway` — Neo4j 查询失败（Bolt 断连 / Cypher 语法等）
- `503 Service Unavailable` — 未配置 `NEO4J_*`

### `GET /kg-viewer`（HTML 页面）

直接打开 `http://<host>:<port>/kg-viewer`：

- 自动 fetch `/v1/graph/kg`，状态条会显示 `已从后端加载 N 节点 / M 关系`
- 失败时回退到内置 4 类型 / 15 节点 / 16 关系的营养示例（Product / Ingredient /
  HealthTopic / Evidence 之间的边），方便无 Neo4j 也能演示
- 控件：自动旋转 / 标签显隐 / 层级间距 / 重置视角 / 从后端加载 / 示例数据 /
  载入 JSON；拖拽 `.json` 文件到窗口也行
- 节点交互：滚轮缩放、拖拽旋转、点击节点聚焦（Product 显示本品及后代，
  其它节点显示可达子图，只保留相关连线）、Esc / 点空白取消

## 测试

```bash
# 单元测试（handler + router + OpenAPI 一致性）
go test -count=1 ./...
#  关键：TestGraphKG_ReturnsSnapshot / _DefaultLimit / _NotConfigured / _LoaderError
#  TestHandler_ServesEmbeddedHTML —— 确认 /kg-viewer 页面嵌入完整
```

## 接入新数据源

快照固定只展示 4 类主节点（`Product` / `Ingredient` / `Evidence` /
`HealthTopic`）。如果想新增/调整展示类型，需要同步改三处：

1. **后端 Cypher 过滤**（`internal/neo4j/client.go`）—— 在
   `WHERE NOT 'EvidenceContext' IN labels(n) AND ...` 里把新类型
   加进允许集合（当前实现是反向黑名单，新类型默认可达）。
2. **后端类型列表** —— 当前 4 类写死在 Cypher；如要扩展"必须包含"的
   白名单，把 `WHERE` 改成 `labels(n) IN $allowedTypes` 并在 Go 端
   注入白名单参数。
3. **前端常量**（`internal/viewer/index.html`）——
   `TYPE_DEPTHS` / `TYPE_COLORS` / `TYPE_LABELS_CN` / `LAYER_RADII` /
   `LAYER_SIZES` / `SAMPLE.types` 都得追加对应条目；`viewer_test.go`
   的 banned 列表也要同步放宽。

Neo4j 里灌新数据： `MERGE (:NewType { name: 'xxx' })` + 关系，
重跑 `make neo4j-seed`（idempotent）或手工 cypher。
新增的 Neo4j label 如果不在前端 4 类常量里，会用 hash 色 + label=type
名兜底渲染（自动追加到额外列表）。

## 文件清单

- `internal/neo4j/client.go` — Bolt 客户端 + Cypher 快照查询
- `internal/handler/graph.go` — `GET /v1/graph/kg` handler
- `internal/handler/graph_test.go` — 4 个 handler 单测
- `internal/viewer/{viewer.go,index.html}` — 嵌入式 3D 页面
- `cmd/server/main.go` — 路由挂载 + 启动期按 env 决定是否构造 `neo4j.Client`
- `internal/config/config.go` — `NEO4J_*` env 绑定
- `docs/openapi.yaml` — `Graph` tag + `KnowledgeGraph` schema + `BadGateway` response
- `deploy/neo4j/docker-compose.yml` + `deploy/neo4j/seeds/seed.cypher` — 本地库
- `Makefile` — `neo4j-up/seed/down/logs` 四个目标

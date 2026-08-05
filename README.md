# kg-viewer

3D 知识图谱可视化服务。从 Neo4j 读取图谱快照，通过 Three.js
进行三维渲染，支持节点聚焦、层级分离、自动旋转等交互。

## 快速开始

```bash
# 1. 配置环境变量
cp .env.example .env

# 2. 启动服务（无需 Neo4j 即可查看 3D 演示）
make run
# → http://localhost:8090/kg-viewer
```

## 接入 Neo4j

```bash
# 启动本地 Neo4j 5.26 Community
make neo4j-up

# 灌入示例营养知识图谱（幂等）
make neo4j-seed

# 浏览器管理台 http://localhost:7474 (neo4j / healthdinner123)
```

`.env` 中配置连接信息（留空 `NEO4J_URI` 则 `/v1/graph/kg` 返回 503，
3D 页面自动回退内置示例数据）：

```bash
NEO4J_URI=bolt://127.0.0.1:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=healthdinner123
NEO4J_DATABASE=neo4j
```

## API

### `GET /v1/graph/kg`

返回图谱快照（节点 + 关系），无鉴权。

| 参数 | 类型 | 默认 | 说明 |
| --- | --- | --- | --- |
| `limit` | int ≥ 1 | 0（全量） | 单关系类型采样上限 |
| `label` | string | "" | 按节点标签过滤（如 `Ingredient`） |

固定 4 类节点：`Product` / `Ingredient` / `Evidence` / `HealthTopic`。
`EvidenceContext` 作为隐式桥（不出现）通过两跳穿透产生
`EVIDENCE_FOR` 和 `SUPPORTS_TOPIC` 虚拟边。

### `GET /kg-viewer`

内嵌的 3D 可视化页面（静态 HTML，`//go:embed` 打入二进制）。

## 架构

```
浏览器 /kg-viewer (Three.js)
  └─ fetch /v1/graph/kg      ← 同源
      └─ neo4j.Client.Fetch  ← Bolt 7687
          └─ Neo4j 5.26
```

## 开发

```bash
make build     # 编译到 bin/kg-viewer
make test      # 单元测试
```

详细文档见 [docs/kg-3d-viewer.md](docs/kg-3d-viewer.md)。

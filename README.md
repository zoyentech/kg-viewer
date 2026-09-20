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

# 浏览器管理台 http://localhost:7474 (neo4j / neo4j-healthdinner-2026)
```

`.env` 中配置连接信息（留空 `NEO4J_URI` 则 `/v1/graph/kg` 返回 503，
3D 页面自动回退内置示例数据）：

场景 A — 本地 Docker 示例：
```bash
NEO4J_URI=bolt://127.0.0.1:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=neo4j-healthdinner-2026
NEO4J_DATABASE=neo4j
```

场景 B1 — M3 英文标签库（Product / Ingredient / Evidence / HealthTopic）：
```bash
NEO4J_URI=bolt://192.168.31.201:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=neo4j-healthdinner-2026
NEO4J_DATABASE=neo4j
```

场景 B2 — M3 中文标签库（配方 / 成分 / 循证证据 / 健康结局）：
```bash
NEO4J_URI=bolt://192.168.31.201:7688
NEO4J_USER=neo4j
NEO4J_PASSWORD=neo4j-healthdinner-2026
NEO4J_DATABASE=neo4j
DB_LANG=zh
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

## 生产部署（ssh prod，端口 8090）

本地配置好 `ssh prod` 后，直接执行：

```bash
make deployprod
```

该命令会交叉编译 Linux/amd64 二进制，通过 `rsync` 上传到
`/opt/kg-viewer`，安装并更新 `kg-viewer.service`（systemd，开机自启），
以 `kgviewer` 用户运行在 `8090` 端口，不配置域名和 nginx，并在重启后
自动健康检查 `http://127.0.0.1:8090/kg-viewer`。

如需在生产接入 Neo4j，把生产环境变量放到仓库根目录 `.env.prod`
（已加入 `.gitignore`），`make deployprod` 会自动上传为
`/opt/kg-viewer/.env`；未提供时保留服务器已有配置，默认以内嵌快照运行。

查看运行状态：

```bash
make deployprod-status
```

### 临时客户演示链接（防爬取）

在 `.env.prod` 中配置一个随机密钥：

```bash
DEMO_SECRET=<openssl rand -hex 32 生成的值>
```

部署后生成限时演示链接：

```bash
make demo-url HOURS=72
```

生成的链接格式为
`http://82.156.88.47:8090/kg-viewer?token=<过期时间>.<HMAC签名>`。
配置了 `DEMO_SECRET` 后，页面、图谱 API 和 `snapshot.json` 都必须携带
有效且未过期的令牌，否则统一返回 404。任何人拿到该链接都能在有效期内
正常查看数据，这是防止无令牌爬虫抓取，不是防止拿到链接的人保存数据。

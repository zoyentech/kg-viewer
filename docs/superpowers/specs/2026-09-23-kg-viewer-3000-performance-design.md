# 3D 知识图谱 3000+ 节点性能优化设计

## 目标

在不改变现有四类节点、点击聚焦、谱系过滤、信息卡和离线快照行为的前提下，让查看器能够稳定承载约 2871 个节点（配方 1000、成分 1458、循证证据 319、健康结局 94），并为未来 3000+ 节点保留余量。

## 当前实现与瓶颈

当前工作区已有主体球体 `THREE.InstancedMesh` 的未提交改造，但仍存在以下放大项：

- 动画循环对所有节点每帧调用 `setMatrixAt`、`setColorAt`，即使节点已经静止；这会持续上传完整实例矩阵和颜色缓冲。
- 每条关系仍然创建独立的 `THREE.Line`、`BufferGeometry` 和 `LineBasicMaterial`。
- 每个节点仍然创建一个透明点击 hitbox、一个 halo Sprite 和一个文字 Sprite。
- hover 每帧对整个节点组做 raycast；标签每帧计算相机距离并更新缩放。
- 32×24 的球体几何对数千节点重复绘制，远距离下的几何细节没有收益。

## 设计

### 1. 节点渲染层

保留一个共享球体 `nodeInstances`，但把节点状态分成不可变布局状态和可变视觉状态：

```text
NodeRecord {
  id, type, position, baseColor, baseSize,
  instanceIndex, hitboxIndex,
  amountShare, renderScale, renderColor,
  instanceVisible, labelSprite
}
```

所有实例只有在以下事件发生时才写入 GPU：入场动画仍在进行、焦点/隐藏类型变化、配方含量比例变化、窗口尺寸/相机变化影响标签时。静止状态不再每帧上传矩阵和颜色。隐藏实例继续使用极小缩放，并在拾取映射中用 `instanceVisible` 过滤。

主体球体使用较低分段的共享几何（16×12），将每节点三角形成本降到当前的约四分之一；实例拾取继续通过 `intersection.instanceId` 反查节点 ID。

### 2. 批量点击层

用第二个共享 `InstancedMesh` 表示透明 hitbox。它只服务于 raycast，不参与可见渲染，所有节点共用一个低分段球体和透明材质。这样点击范围保留，但从约 2871 个对象降为一个对象。映射表独立于主体实例，避免主体尺寸随含量比例变化时影响点击容错范围。

### 3. 关系渲染层

用一个 `THREE.LineSegments` 替代逐边 Line：

- `position`：每条关系两个端点，共享静态 BufferGeometry。
- `color`：两个端点的节点基色。
- `lineOpacity`：两个端点相同的可见性属性。
- `lineBirth`：两个端点相同的入场时间属性。

自定义 shader 使用 `lineOpacity` 实现聚焦/隐藏状态，使用 `uTime - lineBirth` 实现淡入。这样关系状态变化只更新属性，动画不再遍历并修改每条线的材质。

### 4. 配方含量视觉编码

点击一个 `Product` 时，仅对该配方直接连接的 `Ingredient` 计算含量比例：

1. 解析 `properties.amount` 中的首个数值和单位。
2. 将 `g/kg/mg/mcg/μg` 统一到 mg；无法识别单位时使用数值但只在同一组中比较。
3. 对有效含量求 `share = amount / sum(amount)`，再用 `sqrt(share / maxShare)` 压缩极端差异。
4. 成分尺寸使用 `baseSize × (0.72 + 1.35 × normalized)`。
5. 成分颜色从类型基色向白色高亮插值，含量越高亮度越高。

无含量或无法解析的成分保持基色和默认大小；选择非配方节点、取消聚焦或加载新图时恢复默认视觉状态。信息卡仍显示原始 amount 文本，不把视觉归一化结果替换为业务数据。

### 5. 标签 LOD 与拾取节流

标签保留 Sprite 以维持中文字形质量，但增加预算：节点数超过 900 时，只显示焦点、悬停、搜索命中和按相机距离/节点层级排序的前 320 个标签；节点数较少时保持现有全量标签。标签 Sprite 按需创建并在候选集合变化时同步，避免一次性为 3000+ 节点创建 CanvasTexture。

hover raycast 仅在指针移动、相机变化或焦点状态变化后执行，并限制到约 30Hz；点击仍立即执行一次 raycast。这样自动旋转时不会每个渲染帧重复完整拾取。

### 6. 自适应渲染质量与诊断

加载图后根据节点数设置 DPR：小图保留最多 2，3000+ 节点限制到 1.5。页面增加调试统计 `renderer.info.render.calls / triangles / lines` 与可选的 `performance` 日志，验证优化不是只减少 JavaScript 对象而实际 draw call 仍然增长。

## 正确性约束

- 四类节点层级、颜色、标签、搜索、信息卡和谱系 BFS 语义不变。
- 主体节点和 hitbox 两条实例映射均必须过滤不可见节点，不能因为 Raycaster 不检查 `visible` 而穿透到聚焦暗区。
- 关系线隐藏必须使用真正的 opacity 属性，不能只把颜色变暗。
- 动态实例矩阵/颜色更新后必须设置对应 `needsUpdate`；布局改变后更新包围球或明确关闭对象级视锥裁剪。
- 页面仍使用现有 Three.js 0.160.0 CDN 版本，不引入新运行时依赖。

## 验证方案

- Go 单元测试：页面嵌入、批处理结构、amount 解析、实例拾取映射和禁止逐对象关系/主体网格回归。
- 静态模块语法检查：提取 inline module 后用 Node 语法检查。
- 构建与 race 测试：`go test -count=1 -race ./...`、`make build`、`git diff --check`。
- 浏览器验证：加载离线快照和 live API，确认 2871/3000+ 规模下页面无 console error，点击配方后成分大小/颜色变化，Esc/重置后恢复，搜索和 hover 正常。
- 性能证据：记录加载耗时、内存对象数量、`renderer.info` draw calls/triangles/lines，并与当前基线对比。

## 不在本次范围

- 不修改 Neo4j 查询和 API 数据契约。
- 不把所有文字改成复杂的 GPU 字体图集；本次先用标签预算解决主要规模问题。
- 不新增第三方依赖或改变部署方式。

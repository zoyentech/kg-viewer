package viewer

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestHandler_ServesEmbeddedHTML guards against the //go:embed path or
// the route wiring silently breaking. The page hosts the live
// /v1/graph/kg call, so a non-200 here means the demo is dead.
func TestHandler_ServesEmbeddedHTML(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/kg-viewer", Handler)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/kg-viewer", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Fatalf("content-type=%q", got)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`<title>BeauZenith｜营养健康数字具身智能知识图谱</title>`,
		`<svg class="brand-mark"`,
		`<span class="brand-name">BeauZenith</span>`,
		`<span class="brand-subtitle">营养健康数字具身智能知识图谱</span>`,
		`/v1/graph/kg`,               // live API URL
		"three@0.160.0",              // three.js CDN version pin
		"focus && n.id === focus.id", // animation loop must not deref null focus
		"function pickNodeId()",      // click raycast path is wired up
<<<<<<< HEAD
		"new THREE.InstancedMesh",    //主体球体按实例批处理
		"nodeInstances.instanceMatrix.setUsage(THREE.DynamicDrawUsage)",
		"intersection.instanceId",  //实例拾取映射回节点 ID
		"nodeInstances.setColorAt", //实例级别保留节点颜色/焦点高亮
		"raycaster.intersectObject(nodeInstances, false)",
		"new THREE.LineSegments",     // relationships share one draw object
		"linkGeometry.setAttribute('position'",
		"linkColorAttr = new THREE.BufferAttribute", //关系颜色统一存放在批量 BufferGeometry
		"linkGeometry.setAttribute('lineOpacity'",
		"linkOpacityAttr.needsUpdate = true", //逐边透明度由 BufferAttribute 驱动
		"hitboxInstances",            // transparent picking is batched
		"lineOpacity",                // shader-backed per-edge visibility
		"lineBirth",                  // shader-backed edge fade-in
		"parseAmountMg",               // amount normalization for recipe visuals
		"applyRecipeAmountVisuals",   // selected recipe drives ingredient styling
		"MAX_VISIBLE_LABELS",         // large graphs use label LOD
		"nodeVisualsDirty",           // steady-state instances are not uploaded each frame
		"Product: 0,",                // 4-layer: Product → Ingredient → Evidence → HealthTopic
		"Evidence: 2,",               // Evidence 居中（介于 Ingredient 和 HealthTopic 之间）
		// picking 必须自己按 o.visible 过滤：
		//   Three.js Raycaster.intersectObjects() 不检查 object.visible，
		//   focus 模式下隐藏的节点 / hitbox 若不过滤，点击会"穿透"到 lineage 外的暗区。
		"o.visible && o.userData.isHitbox", // pickNodeId() hitbox 过滤
		"o.visible), false",                // 动画循环 hover 过滤
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("viewer page missing %q", want)
		}
	}
	// 防御性断言：EvidenceContext 不应再出现在页面常量里。
	// 后端 Cypher 也过滤掉，剩这一道防线挡住未来重新引入。
	for _, banned := range []string{
		"EvidenceContext: 1,",
		"EvidenceContext: '#ffd166'",
		"evidencecontext 文献检索",
		"new THREE.Line(",             // relationships must be batched
		"new THREE.SphereGeometry(1, 12, 8)", // hitboxes must be batched
		"linkObjs.push",              // no per-link object registry
		"nodeInstances.instanceMatrix.needsUpdate = true;\n    if (nodeInstances.instanceColor) nodeInstances.instanceColor.needsUpdate = true;", // no unconditional upload
		// 防止有人"优化"成不带 visible 过滤的版本（会让 lineage 暗区被穿透点击）
		"nodeGroup.children.filter(o => o.userData.nodeId && !o.userData.isHitbox), false",
		"nodeGroup.children.filter(o => o.userData.isHitbox), false",
		"raycaster.intersectObjects(nodeGroup.children, false)",
		"const line = new THREE.Line(geo, mat);", //关系边不得再逐条创建 Line
		"linkObjs.push({ line, mat,",             //关系元数据不得绑定独立 Line/Material
	} {
		if strings.Contains(body, banned) {
			t.Fatalf("viewer page must not reference %q (4-type filter regression)", banned)
		}
	}
}

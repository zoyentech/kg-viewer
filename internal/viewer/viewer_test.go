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
		"hitboxInstances",            // transparent picking is batched
		"new THREE.InstancedMesh",    //主体球体按实例批处理
		"nodeInstances.instanceMatrix.setUsage(THREE.DynamicDrawUsage)",
		"intersection.instanceId",    //实例拾取映射回节点 ID
		"nodeInstances.setColorAt",   //实例级别保留节点颜色/焦点高亮
		"raycaster.intersectObject(nodeInstances, false)",
		"new THREE.LineSegments",     // relationships share one draw object
		"lineOpacity",                // shader-backed per-edge visibility
		"lineBirth",                  // shader-backed edge fade-in
		"parseAmountMg",              // amount normalization for recipe visuals
		"applyRecipeAmountVisuals",   // selected recipe drives ingredient styling
		"MAX_VISIBLE_LABELS",         // large graphs use label LOD
		"nodeVisualsDirty",           // steady-state instances are not uploaded each frame
		"nodeIdFromIntersection",     // instance picking maps back to graph IDs
		"raycaster.intersectObject(hitboxInstances, false)",
		"if (nodeVisualsDirty && nodeInstances)",
		"window.__KG_VIEWER_DEBUG__",
		"renderer.info.render.calls",
		"const SEARCH_TYPE_ORDER = ['Product', 'Ingredient', 'HealthTopic'];",
		"const allOrdered = SEARCH_TYPE_ORDER.filter(t => counts[t]);",
		"if (!searchState.query) {", //空查询不展示搜索结果下拉
		"linkGeometry.setAttribute('position'",
		"linkGeometry.setAttribute('lineOpacity'",
		"Product: 0,",                // 4-layer: Product → Ingredient → Evidence → HealthTopic
		"Evidence: 2,",               // Evidence 居中（介于 Ingredient 和 HealthTopic 之间）
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
		// 防止有人"优化"成不带 visible 过滤的版本（会让 lineage 暗区被穿透点击）
		"nodeGroup.children.filter(o => o.userData.nodeId && !o.userData.isHitbox), false",
		"nodeGroup.children.filter(o => o.userData.isHitbox), false",
		"raycaster.intersectObjects(nodeGroup.children, false)",
	} {
		if strings.Contains(body, banned) {
			t.Fatalf("viewer page must not reference %q (4-type filter regression)", banned)
		}
	}
}

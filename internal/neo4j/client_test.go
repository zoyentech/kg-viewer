package neo4j

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestFilterDirty_RemovesSentinelNodesAndLinks(t *testing.T) {
	dirtyID := "4:abc:1"
	cleanID := "4:abc:2"
	kg := &KnowledgeGraph{
		Nodes: []GraphNode{
			{ID: dirtyID, Label: "未检索到符合条件的 PubMed 系统综述", Type: "Evidence"},
			{ID: cleanID, Label: "维生素C", Type: "Ingredient"},
		},
		Links: []GraphLink{
			{Source: cleanID, Target: dirtyID, Type: "EVIDENCE_FOR"},
			{Source: dirtyID, Target: cleanID, Type: "SUPPORTS_TOPIC"},
		},
		Source: "neo4j",
	}

	out := FilterDirty(kg)
	if len(out.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(out.Nodes))
	}
	if out.Nodes[0].ID != cleanID {
		t.Fatalf("expected clean node to survive, got %s", out.Nodes[0].ID)
	}
	if len(out.Links) != 0 {
		t.Fatalf("expected 0 links after dropping dirty endpoint, got %d", len(out.Links))
	}
}

func TestV2LabelSetsSelectLanguageSpecificDisplayFields(t *testing.T) {
	if got := V2EnglishLabels.DisplayKeys[0]; got != "name_en" {
		t.Fatalf("v2 English first display key = %q, want name_en", got)
	}
	if got := V2ChineseLabels.DisplayKeys[0]; got != "name_zh" {
		t.Fatalf("v2 Chinese first display key = %q, want name_zh", got)
	}

	m := map[string]any{
		"name_en": "Vitamin A",
		"name_zh": "维生素 A",
	}
	if got := resolveLabel(m, V2EnglishLabels.DisplayKeys); got != "Vitamin A" {
		t.Fatalf("English v2 label = %q, want Vitamin A", got)
	}
	if got := resolveLabel(m, V2ChineseLabels.DisplayKeys); got != "维生素 A" {
		t.Fatalf("Chinese v2 label = %q, want 维生素 A", got)
	}
}

func TestFilterDirty_PassesThroughWhenClean(t *testing.T) {
	kg := &KnowledgeGraph{
		Nodes:  []GraphNode{{ID: "n1", Label: "维生素D", Type: "Ingredient"}},
		Links:  []GraphLink{},
		Source: "snapshot",
	}
	out := FilterDirty(kg)
	// Should return the same pointer when nothing was filtered.
	if out != kg {
		t.Fatal("expected original graph when nothing dirty")
	}
}

func TestFilterDirty_NilSafe(t *testing.T) {
	if out := FilterDirty(nil); out != nil {
		t.Fatal("expected nil passthrough")
	}
}

func TestFilterDirty_DescriptionMatch(t *testing.T) {
	kg := &KnowledgeGraph{
		Nodes: []GraphNode{
			{ID: "x", Label: "some-label", Type: "Evidence", Description: "未检索到符合条件的 PubMed 文献"},
			{ID: "y", Label: "clean", Type: "Ingredient"},
		},
		Links: nil,
	}
	out := FilterDirty(kg)
	if len(out.Nodes) != 1 || out.Nodes[0].ID != "y" {
		t.Fatalf("expected only clean node, got %+v", out.Nodes)
	}
}

// ===========================================================================
// resolveLabel — UTF-8 safe truncation regression tests.
//
// Background: resolveLabel used to do `if len(v) > 40 { return v[:40] }`,
// which truncates by *byte*. Chinese / CJK characters occupy 3 bytes in
// UTF-8, so the byte slice cut a multi-byte sequence in half and the
// browser's JSON.parse replaced the orphan byte with U+FFFD "�". The
// fix truncates by rune (Unicode code-point) instead. These tests guard
// against reintroducing the byte-slice variant in any form.
// ===========================================================================

// TestResolveLabel_ShortChinesePassThrough guards the original bug
// report: a 16-rune Chinese title must be returned intact.
func TestResolveLabel_ShortChinesePassThrough(t *testing.T) {
	m := map[string]any{
		"name":  "核黄素预防偏头痛：一项系统评价。", // 16 runes, 48 bytes
		"title": "核黄素预防偏头痛：一项系统评价。",
	}
	got := resolveLabel(m, []string{"name", "title"})
	if got != "核黄素预防偏头痛：一项系统评价。" {
		t.Fatalf("expected full title, got %q", got)
	}
	if strings.ContainsRune(got, '\uFFFD') {
		t.Fatalf("label must not contain U+FFFD, got %q", got)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("label must be valid UTF-8, got %q", got)
	}
}

// TestResolveLabel_TruncatesAtRuneBoundary constructs a 60-rune mixed
// (CJK + ASCII + colon) string and asserts the result is exactly
// 40 runes, valid UTF-8, and contains no replacement character.
func TestResolveLabel_TruncatesAtRuneBoundary(t *testing.T) {
	// 偏头痛防治：一项系统评价研究。 = 15 runes (3 CJK × 5 + 2 punctuation
	// doesn't quite work out; rebuild the string character by character
	// so the rune count is unambiguous).
	src := strings.Repeat("偏头痛防治研究", 4) + "：系统评价草稿XXX" // 16×4=64 wait, recompute
	// Build 60 runes deterministically: 12 CJK + "：" + 47 CJK
	runes := []rune{}
	for i := 0; i < 12; i++ {
		runes = append(runes, '偏')
	}
	runes = append(runes, '：')
	for i := 0; i < 47; i++ {
		runes = append(runes, '头')
	}
	src = string(runes) // 60 runes total
	if got := len([]rune(src)); got != 60 {
		t.Fatalf("test fixture wrong: want 60 runes, got %d", got)
	}
	m := map[string]any{"title": src}
	got := resolveLabel(m, []string{"title"})

	// 1. Rune count is exactly 40.
	if c := len([]rune(got)); c != 40 {
		t.Errorf("rune count: want 40, got %d (label=%q)", c, got)
	}
	// 2. No U+FFFD anywhere.
	if strings.ContainsRune(got, '\uFFFD') {
		t.Errorf("label must not contain U+FFFD, got %q", got)
	}
	// 3. Strictly valid UTF-8 (utf8.ValidString checks the byte
	//    sequence is well-formed, catching the original byte-slice bug).
	if !utf8.ValidString(got) {
		t.Errorf("label must be valid UTF-8, got %q (bytes=%x)", got, []byte(got))
	}
	// 4. Prefix is the first 40 runes of the input.
	want := string([]rune(src)[:40])
	if got != want {
		t.Errorf("label mismatch:\n want=%q\n got =%q", want, got)
	}
}

// TestResolveLabel_NoFFFDAtByteCutBoundary reproduces the *exact*
// failure mode of the old code: a string whose 40th byte falls inside
// a CJK character. With the bug, v[:40] returned 13 valid 3-byte
// runes + 1 orphan byte (0xE8), which the browser would replace with
// U+FFFD. After the fix, the result must have no replacement chars.
func TestResolveLabel_NoFFFDAtByteCutBoundary(t *testing.T) {
	// 13 CJK chars = 39 bytes; the 40th byte would have been the first
	// byte of a 14th CJK char. Construct exactly this shape:
	const want = "核黄素预防偏头痛：一项系统评价。XYZ" // 16+3 = 19 runes, but 13 CJK = 39 bytes + "评价。XYZ" = 6 bytes more = 45 bytes
	_ = want
	src := "核黄素预防偏头痛：一项系统评价。" + "XYZ" + strings.Repeat("X", 30)
	if len([]byte(src)) <= 40 {
		t.Fatalf("test fixture must be > 40 bytes to trigger truncation")
	}
	m := map[string]any{"title": src}
	got := resolveLabel(m, []string{"title"})

	if strings.ContainsRune(got, '\uFFFD') {
		t.Errorf("the old code would have produced \\uFFFD here; got %q (bytes=%x)", got, []byte(got))
	}
	if !utf8.ValidString(got) {
		t.Errorf("label must be valid UTF-8, got %q", got)
	}
	// And it must equal the first 40 runes of the input.
	runes := []rune(src)
	if string(runes[:40]) != got {
		t.Errorf("expected first-40-rune prefix, got %q", got)
	}
}

// TestResolveLabel_KeyPriority covers the key-fallback list semantics:
// "name" wins over "title" wins over generic fallbacks. Truncation
// applies per the *selected* value, not whichever happens to be > 40
// bytes.
func TestResolveLabel_KeyPriority(t *testing.T) {
	m := map[string]any{
		"name":  "短名",                       // 2 runes
		"title": strings.Repeat("超长标题", 20), // over 40 runes
	}
	got := resolveLabel(m, []string{"name", "title", "label"})
	if got != "短名" {
		t.Errorf("expected first non-empty key to win; got %q", got)
	}
}

// TestResolveLabel_EmptyAndMissing ensures nil/empty values fall
// through to the idn fallback (and finally to ""), without ever
// returning a string that contains U+FFFD.
func TestResolveLabel_EmptyAndMissing(t *testing.T) {
	cases := []struct {
		name string
		m    map[string]any
		keys []string
		want string
	}{
		{"all empty strings", map[string]any{"name": "", "title": ""}, []string{"name", "title"}, ""},
		{"missing keys", map[string]any{}, []string{"name", "title"}, ""},
		{"falls back to idn", map[string]any{"idn": int64(42)}, []string{"name", "title"}, "#42"},
		{"whitespace only is not empty", map[string]any{"name": "   "}, []string{"name"}, "   "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := resolveLabel(c.m, c.keys)
			if got != c.want {
				t.Errorf("want %q, got %q", c.want, got)
			}
			if strings.ContainsRune(got, '\uFFFD') {
				t.Errorf("must not contain U+FFFD, got %q", got)
			}
		})
	}
}

// TestResolveLabel_NoTruncationUnderThreshold is a property-style
// sweep: every title in the [1, 40] rune window must round-trip
// exactly, and the byte length of the result must equal the byte
// length of the input (proving the function does no rewriting).
func TestResolveLabel_NoTruncationUnderThreshold(t *testing.T) {
	for n := 1; n <= 40; n++ {
		src := strings.Repeat("测", n)
		m := map[string]any{"name": src}
		got := resolveLabel(m, []string{"name"})
		if got != src {
			t.Errorf("n=%d: want %q, got %q", n, src, got)
		}
		if len([]byte(got)) != len([]byte(src)) {
			t.Errorf("n=%d: byte length changed from %d to %d", n, len([]byte(src)), len([]byte(got)))
		}
	}
}

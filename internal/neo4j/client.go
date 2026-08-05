// Package neo4j provides a thin read-only client that snapshots a
// knowledge graph (nodes + relationships) for the 3D viewer API.
// It supports both English-labeled (:7687) and Chinese-labeled (:7688)
// Neo4j databases via LabelSet.
package neo4j

import (
	"context"
	"fmt"
	"strconv"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// LabelSet maps the semantic roles used in the Cypher query to actual
// Neo4j label names. EnglishLabels for :7687, ChineseLabels for :7688.
type LabelSet struct {
	EvidenceContext string // EN: "EvidenceContext"  ZH: "证据上下文"
	Evidence        string // EN: "Evidence"          ZH: "循证证据"
	HealthTopic     string // EN: "HealthTopic"       ZH: "健康结局"
}

// EnglishLabels targets the M3 :7687 database (nutrition-evidence-kg).
var EnglishLabels = LabelSet{
	EvidenceContext: "EvidenceContext",
	Evidence:        "Evidence",
	HealthTopic:     "HealthTopic",
}

// ChineseLabels targets the M3 :7688 database (nutrition-evidence-kg-zh).
var ChineseLabels = LabelSet{
	EvidenceContext: "证据上下文",
	Evidence:        "循证证据",
	HealthTopic:     "健康结局",
}

// chineseToEnglish normalises Chinese labels to the canonical English
// type names the viewer expects (TYPE_DEPTHS / TYPE_COLORS etc.).
var chineseToEnglish = map[string]string{
	"配方":       "Product",
	"成分":       "Ingredient",
	"循证证据":     "Evidence",
	"健康结局":     "HealthTopic",
	"证据上下文":    "EvidenceContext",
	"证据层级":     "EvidenceLayer",
}

// GraphNode is a node as exposed by GET /v1/graph/kg (matches the
// payload shape consumed by the 3D viewer).
type GraphNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

// GraphLink is a directed relationship between two node IDs.
type GraphLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type,omitempty"`
}

// KnowledgeGraph is the snapshot payload rendered by the 3D viewer.
type KnowledgeGraph struct {
	Nodes  []GraphNode `json:"nodes"`
	Links  []GraphLink `json:"links"`
	Source string      `json:"source"`
}

// Options controls the snapshot query.
type Options struct {
	// Limit caps the number of relationship rows sampled per
	// relationship type. <=0 means "no cap, return every matching
	// relationship". EvidenceContext is always excluded; the kg-viewer
	// only renders Product / Ingredient / Evidence / HealthTopic.
	Limit int
	// Label restricts the snapshot to nodes carrying this label
	// (e.g. "Ingredient"); empty means the whole graph.
	Label string
}

// Client talks to a single Neo4j database over the official driver.
type Client struct {
	driver neo4j.DriverWithContext
	db     string
	labels LabelSet
}

// NewClient opens a driver and verifies connectivity. labels selects
// the label schema (EnglishLabels or ChineseLabels).
func NewClient(ctx context.Context, uri, user, password, database string, labels LabelSet) (*Client, error) {
	drv, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, password, ""))
	if err != nil {
		return nil, fmt.Errorf("neo4j driver: %w", err)
	}
	if err := drv.VerifyConnectivity(ctx); err != nil {
		_ = drv.Close(ctx)
		return nil, fmt.Errorf("neo4j connectivity: %w", err)
	}
	return &Client{driver: drv, db: database, labels: labels}, nil
}

// Close releases the underlying driver.
func (c *Client) Close(ctx context.Context) error { return c.driver.Close(ctx) }

// Fetch runs a Cypher snapshot over the graph. The kg-viewer renders
// 4 principal types: Product / Ingredient / Evidence / HealthTopic.
// EvidenceContext is used as a hidden bridge (never emitted as a node)
// to materialise two-hop edges:
//
//	Ingredient -(EvidenceContext)- Evidence   (EVIDENCE_FOR)
//	Evidence   -(EvidenceContext)- HealthTopic (SUPPORTS_TOPIC)
//
// so Evidence sits visually *between* Ingredient and HealthTopic
// (Product -> Ingredient -> Evidence -> HealthTopic). Direct edges among
// the 4 types are returned verbatim. Per-node label resolution happens
// in Go so we can handle long titles and label-specific properties
// (canonical_name for Ingredient, name for Product, etc.). When Limit >
// 0 each UNION branch is independently sampled (useful for ad-hoc curl
// debugging); Limit <= 0 returns everything. Label names are taken from
// c.labels so both English (:7687) and Chinese (:7688) databases work.
func (c *Client) Fetch(ctx context.Context, opts Options) (*KnowledgeGraph, error) {
	// UNION 三分支:每行返回 (anode, bnode, rt),Go 端多行循环去重。
	// Limit > 0 时每分支加 LIMIT(整数拼接,安全无注入)。
	// 标签名通过 fmt 拼入(来自 LabelSet 硬编码,非用户输入,无注入风险)。
	limitClause := ""
	if opts.Limit > 0 {
		limitClause = fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	L := c.labels
	labelProps := []string{
		"canonical_name", // Ingredient
		"name",           // Product / HealthTopic
		"title",          // Evidence
		"label",          // generic
		"topic_id",       // HealthTopic fallback
		"evidence_id",    // Evidence fallback
		"product_id",     // Product fallback
		"ingredient_id",  // Ingredient fallback
	}
	// nodeProj 把一个节点展开成 label-resolution 用的 map。
	nodeProj := func(varName string) string {
		return fmt.Sprintf(`{
    id: elementId(%[1]s),
    type: head([l IN labels(%[1]s) WHERE NOT l STARTS WITH '_']),
    canonical_name: %[1]s.canonical_name,
    name: %[1]s.name,
    title: %[1]s.title,
    label: %[1]s.label,
    topic_id: %[1]s.topic_id,
    evidence_id: %[1]s.evidence_id,
    product_id: %[1]s.product_id,
    ingredient_id: %[1]s.ingredient_id,
    idn: id(%[1]s)
  }`, varName)
	}
	query := fmt.Sprintf(`
// 分支 1:4 类节点间的直连边(排除 EvidenceContext)
MATCH (a)-[r]->(b)
WHERE NOT '%[1]s' IN labels(a) AND NOT '%[1]s' IN labels(b)
  AND ($label = '' OR $label IN labels(a) OR $label IN labels(b))
RETURN %[2]s AS anode, %[3]s AS bnode, type(r) AS rt%[4]s
UNION ALL
// 分支 2:Ingredient -(EC)- Evidence  (EVIDENCE_FOR)
MATCH (ing)-[]-(ec1)-[]-(ev)
WHERE '%[1]s' IN labels(ec1) AND '%[5]s' IN labels(ev)
  AND ing <> ev AND NOT '%[1]s' IN labels(ing)
  AND ($label = '' OR $label IN labels(ing) OR $label IN labels(ev))
RETURN %[6]s AS anode, %[7]s AS bnode, 'EVIDENCE_FOR' AS rt%[4]s
UNION ALL
// 分支 3:Evidence -(EC)- HealthTopic  (SUPPORTS_TOPIC)
MATCH (ev2)-[]-(ec2)-[]-(ht)
WHERE '%[5]s' IN labels(ev2) AND '%[1]s' IN labels(ec2) AND '%[8]s' IN labels(ht)
  AND ev2 <> ht
  AND ($label = '' OR $label IN labels(ev2) OR $label IN labels(ht))
RETURN %[9]s AS anode, %[10]s AS bnode, 'SUPPORTS_TOPIC' AS rt%[4]s
`,
		L.EvidenceContext,                              // %[1]s
		nodeProj("a"), nodeProj("b"), limitClause,     // %[2]s %[3]s %[4]s
		L.Evidence,                                    // %[5]s
		nodeProj("ing"), nodeProj("ev"),               // %[6]s %[7]s
		L.HealthTopic,                                 // %[8]s
		nodeProj("ev2"), nodeProj("ht"),               // %[9]s %[10]s
	)

	sess := c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: c.db,
		AccessMode:   neo4j.AccessModeRead,
	})
	defer sess.Close(ctx)

	res, err := sess.Run(ctx, query, map[string]any{"label": opts.Label})
	if err != nil {
		return nil, fmt.Errorf("neo4j query: %w", err)
	}

	// 多行循环:每行一对节点 + 关系类型。Go 端统一去重。
	nodes := make([]GraphNode, 0, 2048)
	seenNode := make(map[string]bool, 2048)
	links := make([]GraphLink, 0, 4096)
	seenLink := make(map[string]bool, 4096)

	addNode := func(m map[string]any) {
		id := str(m["id"])
		if id == "" || seenNode[id] {
			return
		}
		seenNode[id] = true
		typ := str(m["type"])
		// Normalise Chinese labels to canonical English type names so
		// the viewer's TYPE_DEPTHS / TYPE_COLORS match up.
		if en, ok := chineseToEnglish[typ]; ok {
			typ = en
		}
		if typ == "" {
			typ = "node"
		}
		nodes = append(nodes, GraphNode{
			ID:    id,
			Label: resolveLabel(m, labelProps),
			Type:  typ,
		})
	}

	for res.Next(ctx) {
		rec := res.Record()
		aMap, _ := rec.Values[0].(map[string]any)
		bMap, _ := rec.Values[1].(map[string]any)
		rt := str(rec.Values[2])
		if aMap != nil {
			addNode(aMap)
		}
		if bMap != nil {
			addNode(bMap)
		}
		src, dst := str(aMap["id"]), str(bMap["id"])
		if src == "" || dst == "" || src == dst {
			continue
		}
		key := src + "\x00" + dst + "\x00" + rt
		if seenLink[key] {
			continue
		}
		seenLink[key] = true
		links = append(links, GraphLink{Source: src, Target: dst, Type: rt})
	}
	if err := res.Err(); err != nil {
		return nil, fmt.Errorf("neo4j read: %w", err)
	}

	return &KnowledgeGraph{Nodes: nodes, Links: links, Source: "neo4j"}, nil
}

// resolveLabel picks the best human-readable string for a node, with
// 40-char truncation so the 3D label sprites stay legible. Falls back
// to the numeric internal id when nothing else is set.
func resolveLabel(m map[string]any, keys []string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			if len(v) > 40 {
				return v[:40]
			}
			return v
		}
	}
	if id, ok := m["idn"]; ok {
		return fmt.Sprintf("#%v", id)
	}
	return ""
}

// str normalises Neo4j property values (string / number / bool / nil)
// into a stable display string.
func str(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", t)
	}
}

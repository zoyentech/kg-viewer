package neo4j

import (
	"testing"
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

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zoyentech/kg-viewer/internal/neo4j"
)

// SnapshotLoader serves a pre-cached Neo4j snapshot from JSON bytes.
// It implements both GraphLoader and NodeDetailer so it can function
// as a standalone offline data source or as a fallback when Neo4j
// is unreachable (e.g. developing away from the M3 server).
type SnapshotLoader struct {
	kg   *neo4j.KnowledgeGraph
	byID map[string]neo4j.GraphNode
}

// NewSnapshotLoader parses raw snapshot JSON into an in-memory graph.
func NewSnapshotLoader(raw []byte) (*SnapshotLoader, error) {
	var kg neo4j.KnowledgeGraph
	if err := json.Unmarshal(raw, &kg); err != nil {
		return nil, fmt.Errorf("parse snapshot: %w", err)
	}
	sl := &SnapshotLoader{kg: &kg, byID: make(map[string]neo4j.GraphNode, len(kg.Nodes))}
	for _, n := range kg.Nodes {
		sl.byID[n.ID] = n
	}
	return sl, nil
}

// Fetch returns the cached graph. When opts.Label is set, nodes are
// filtered to those whose Type matches (case-insensitive).
func (sl *SnapshotLoader) Fetch(_ context.Context, opts neo4j.Options) (*neo4j.KnowledgeGraph, error) {
	if opts.Label == "" {
		result := *sl.kg
		result.Source = "snapshot"
		return neo4j.FilterDirty(&result), nil
	}
	want := strings.ToLower(opts.Label)
	keep := make(map[string]bool, len(sl.kg.Nodes))
	var nodes []neo4j.GraphNode
	for _, n := range sl.kg.Nodes {
		if strings.ToLower(n.Type) == want {
			keep[n.ID] = true
			nodes = append(nodes, n)
		}
	}
	var links []neo4j.GraphLink
	for _, l := range sl.kg.Links {
		if keep[l.Source] && keep[l.Target] {
			links = append(links, l)
		}
	}
	return neo4j.FilterDirty(&neo4j.KnowledgeGraph{Nodes: nodes, Links: links, Source: "snapshot"}), nil
}

// NodeDetail looks up a single node by elementId from the cache.
func (sl *SnapshotLoader) NodeDetail(_ context.Context, elementID string) (*neo4j.NodeDetailResult, error) {
	n, ok := sl.byID[elementID]
	if !ok {
		return nil, fmt.Errorf("node not found in snapshot: %s", elementID)
	}
	return &neo4j.NodeDetailResult{
		ID:          n.ID,
		Type:        n.Type,
		Description: n.Description,
		Properties:  n.Properties,
	}, nil
}

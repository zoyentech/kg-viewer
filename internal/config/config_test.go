package config

import "testing"

func TestLoad_V2ChineseProfileUsesBilingualInstance(t *testing.T) {
	t.Setenv("KG_GRAPH_PROFILE", "v2-zh")
	t.Setenv("KG_GRAPH_V2_URI", "bolt://192.168.31.201:7689")
	t.Setenv("KG_GRAPH_V2_DATABASE", "neo4j")
	t.Setenv("NEO4J_URI", "bolt://unused:7687")
	t.Setenv("NEO4J_DATABASE", "unused")
	t.Setenv("DB_LANG", "en")

	cfg := Load()
	if cfg.Neo4jURI != "bolt://192.168.31.201:7689" {
		t.Fatalf("Neo4jURI = %q, want v2 URI", cfg.Neo4jURI)
	}
	if cfg.Neo4jDatabase != "neo4j" {
		t.Fatalf("Neo4jDatabase = %q, want neo4j", cfg.Neo4jDatabase)
	}
	if !cfg.IsChinese() {
		t.Fatal("v2-zh profile should select Chinese display fields")
	}
	if !cfg.IsBilingual() {
		t.Fatal("v2 profile should select bilingual schema")
	}
}

func TestLoad_V3ChineseProfileUsesOutcomeInstance(t *testing.T) {
	t.Setenv("KG_GRAPH_PROFILE", "v3-zh")
	t.Setenv("KG_GRAPH_V3_URI", "bolt://192.168.31.201:7689")
	t.Setenv("KG_GRAPH_V3_DATABASE", "neo4j")
	t.Setenv("NEO4J_URI", "bolt://unused:7687")
	t.Setenv("NEO4J_DATABASE", "unused")
	t.Setenv("DB_LANG", "en")

	cfg := Load()
	if cfg.Neo4jURI != "bolt://192.168.31.201:7689" || cfg.Neo4jDatabase != "neo4j" {
		t.Fatalf("v3 config = (%q, %q), want v3 instance", cfg.Neo4jURI, cfg.Neo4jDatabase)
	}
	if !cfg.IsChinese() || !cfg.IsBilingual() {
		t.Fatal("v3-zh profile should select bilingual Chinese display fields")
	}
}

func TestLoad_LegacyChineseProfileUsesLegacyInstance(t *testing.T) {
	t.Setenv("KG_GRAPH_PROFILE", "legacy-zh")
	t.Setenv("KG_GRAPH_LEGACY_ZH_URI", "bolt://192.168.31.201:7688")
	t.Setenv("KG_GRAPH_LEGACY_ZH_DATABASE", "neo4j")
	t.Setenv("NEO4J_URI", "bolt://unused:7687")

	cfg := Load()
	if cfg.Neo4jURI != "bolt://192.168.31.201:7688" {
		t.Fatalf("Neo4jURI = %q, want legacy Chinese URI", cfg.Neo4jURI)
	}
	if !cfg.IsChinese() {
		t.Fatal("legacy-zh profile should select Chinese labels")
	}
	if cfg.IsBilingual() {
		t.Fatal("legacy profile should not select bilingual schema")
	}
}

func TestLoad_WithoutProfilePreservesLegacyEnvironmentBehavior(t *testing.T) {
	t.Setenv("KG_GRAPH_PROFILE", "")
	t.Setenv("NEO4J_URI", "bolt://legacy:7687")
	t.Setenv("NEO4J_DATABASE", "neo4j")
	t.Setenv("DB_LANG", "zh")

	cfg := Load()
	if cfg.Neo4jURI != "bolt://legacy:7687" || !cfg.IsChinese() || cfg.IsBilingual() {
		t.Fatalf("legacy env behavior not preserved: %+v", cfg)
	}
}

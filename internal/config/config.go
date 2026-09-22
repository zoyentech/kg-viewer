// Package config loads kg-viewer settings from environment variables.
package config

import "os"

// Config holds all runtime configuration for the kg-viewer service.
type Config struct {
	Port          string
	GraphProfile  string
	GraphLanguage string
	Bilingual     bool
	Neo4jURI      string
	Neo4jUser     string
	Neo4jPassword string
	Neo4jDatabase string
	DemoSecret    string
}

// Load reads configuration from the environment. NEO4J_URI empty
// means the /v1/graph/kg endpoint stays in 503 "not configured" state;
// the 3D viewer page falls back to built-in sample data.
func Load() *Config {
	profile := os.Getenv("KG_GRAPH_PROFILE")
	language := os.Getenv("DB_LANG")
	bilingual := false
	uri := os.Getenv("NEO4J_URI")
	database := getEnv("NEO4J_DATABASE", "neo4j")

	// Profiles make the graph version and language a single root-config
	// switch. Legacy Community instances each expose their own `neo4j`
	// database; v2 uses one bilingual database on its own instance.
	switch profile {
	case "legacy-en":
		uri = getEnv("KG_GRAPH_LEGACY_EN_URI", uri)
		database = getEnv("KG_GRAPH_LEGACY_EN_DATABASE", database)
		language = "en"
	case "legacy-zh":
		uri = getEnv("KG_GRAPH_LEGACY_ZH_URI", uri)
		database = getEnv("KG_GRAPH_LEGACY_ZH_DATABASE", database)
		language = "zh"
	case "v2-en":
		uri = getEnv("KG_GRAPH_V2_URI", uri)
		database = getEnv("KG_GRAPH_V2_DATABASE", database)
		language = "en"
		bilingual = true
	case "v2-zh":
		uri = getEnv("KG_GRAPH_V2_URI", uri)
		database = getEnv("KG_GRAPH_V2_DATABASE", database)
		language = "zh"
		bilingual = true
	}

	return &Config{
		Port:          getEnv("KG_VIEWER_PORT", "8090"),
		GraphProfile:  profile,
		GraphLanguage: language,
		Bilingual:     bilingual,
		Neo4jURI:      uri,
		Neo4jUser:     os.Getenv("NEO4J_USER"),
		Neo4jPassword: os.Getenv("NEO4J_PASSWORD"),
		Neo4jDatabase: database,
		DemoSecret:    os.Getenv("DEMO_SECRET"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// IsChinese returns true when the selected graph profile uses Chinese fields.
func (c *Config) IsChinese() bool { return c.GraphLanguage == "zh" }

// IsBilingual returns true for v2, where language is selected through
// bilingual properties instead of a separate Chinese label schema.
func (c *Config) IsBilingual() bool { return c.Bilingual }

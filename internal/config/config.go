// Package config loads kg-viewer settings from environment variables.
package config

import "os"

// Config holds all runtime configuration for the kg-viewer service.
type Config struct {
	Port          string
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
	return &Config{
		Port:          getEnv("KG_VIEWER_PORT", "8090"),
		Neo4jURI:      os.Getenv("NEO4J_URI"),
		Neo4jUser:     os.Getenv("NEO4J_USER"),
		Neo4jPassword: os.Getenv("NEO4J_PASSWORD"),
		Neo4jDatabase: getEnv("NEO4J_DATABASE", "neo4j"),
		DemoSecret:    os.Getenv("DEMO_SECRET"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// IsChinese returns true when DB_LANG=zh (M3 Chinese-labeled KG).
func (c *Config) IsChinese() bool { return os.Getenv("DB_LANG") == "zh" }

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/zoyentech/kg-viewer/internal/config"
	"github.com/zoyentech/kg-viewer/internal/neo4j"
)

func main() {
	cfg := config.Load()
	if cfg.Neo4jURI == "" {
		fmt.Fprintln(os.Stderr, "NEO4J_URI not set")
		os.Exit(1)
	}

	labels := neo4j.EnglishLabels
	if cfg.IsChinese() {
		labels = neo4j.ChineseLabels
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := neo4j.NewClient(ctx, cfg.Neo4jURI, cfg.Neo4jUser, cfg.Neo4jPassword, cfg.Neo4jDatabase, labels)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect:", err)
		os.Exit(1)
	}
	defer client.Close(ctx)

	kg, err := client.Fetch(ctx, neo4j.Options{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "fetch:", err)
		os.Exit(1)
	}

	outPath := "internal/viewer/snapshot.json"
	if len(os.Args) > 1 {
		outPath = os.Args[1]
	}
	abs, _ := filepath.Abs(outPath)

	raw, err := json.MarshalIndent(kg, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal:", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outPath, raw, 0644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}

	fmt.Printf("snapshot saved: %s (%d nodes, %d links, %.1f KB)\n",
		abs, len(kg.Nodes), len(kg.Links), float64(len(raw))/1024)
}

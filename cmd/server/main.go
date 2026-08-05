package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zoyentech/kg-viewer/internal/config"
	"github.com/zoyentech/kg-viewer/internal/handler"
	"github.com/zoyentech/kg-viewer/internal/neo4j"
	"github.com/zoyentech/kg-viewer/internal/viewer"
	"github.com/zoyentech/kg-viewer/pkg/logger"
)

func main() {
	cfg := config.Load()
	log := logger.New()

	// Optional Neo4j-backed knowledge graph. The /v1/graph/kg endpoint
	// answers 503 until NEO4J_URI is set; the viewer page falls back
	// to built-in sample data.
	var graphLoader handler.GraphLoader
	if cfg.Neo4jURI != "" {
		labels := neo4j.EnglishLabels
		if cfg.IsChinese() {
			labels = neo4j.ChineseLabels
			log.Infof("Using Chinese label schema (DB_LANG=zh)")
		}
		nclient, err := neo4j.NewClient(
			context.Background(), cfg.Neo4jURI, cfg.Neo4jUser, cfg.Neo4jPassword, cfg.Neo4jDatabase, labels)
		if err != nil {
			log.Fatalf("neo4j: %v", err)
		}
		defer nclient.Close(context.Background())
		graphLoader = nclient
		log.Infof("Neo4j knowledge graph enabled (db=%s)", cfg.Neo4jDatabase)
	} else {
		log.Infof("Neo4j not configured — /v1/graph/kg returns 503, viewer uses sample data")
	}

	r := gin.Default()

	// 3D knowledge-graph viewer (embedded static page, same origin as API).
	r.GET("/kg-viewer", viewer.Handler)
	r.GET("/kg-viewer/", viewer.Handler)

	// Knowledge-graph snapshot API.
	gh := handler.NewGraphHandler(graphLoader, log)
	r.GET("/v1/graph/kg", gh.KG)
	r.GET("/v1/node/detail", gh.NodeDetail)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Infof("kg-viewer listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Errorf("shutdown: %v", err)
	}
}

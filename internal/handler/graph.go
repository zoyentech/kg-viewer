package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zoyentech/kg-viewer/internal/neo4j"
	"github.com/zoyentech/kg-viewer/pkg/logger"
	"github.com/zoyentech/kg-viewer/pkg/response"
)

// GraphLoader reads a knowledge-graph snapshot. Implemented by
// *neo4j.Client; tests use a stub.
type GraphLoader interface {
	Fetch(ctx context.Context, opts neo4j.Options) (*neo4j.KnowledgeGraph, error)
}

// GraphHandler exposes the knowledge graph consumed by the 3D viewer.
type GraphHandler struct {
	loader GraphLoader
	log    logger.Logger
}

// NewGraphHandler builds the handler. loader may be nil — the endpoint
// then answers 503 until Neo4j is configured (see NEO4J_*).
func NewGraphHandler(loader GraphLoader, log logger.Logger) *GraphHandler {
	return &GraphHandler{loader: loader, log: log}
}

// KG handles GET /v1/graph/kg — returns the node/link snapshot for the
// 3D knowledge-graph viewer.
func (h *GraphHandler) KG(c *gin.Context) {
	if h.loader == nil {
		response.Error(c, http.StatusServiceUnavailable, "Neo4j 未配置：设置 NEO4J_URI/USER/PASSWORD 后重启")
		return
	}
	opts := neo4j.Options{Label: c.Query("label")}
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			opts.Limit = n
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	g, err := h.loader.Fetch(ctx, opts)
	if err != nil {
		h.log.Errorf("kg fetch: %v", err)
		response.Error(c, http.StatusBadGateway, "Neo4j 查询失败: "+err.Error())
		return
	}
	response.OK(c, g)
}

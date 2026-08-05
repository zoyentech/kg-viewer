package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zoyentech/kg-viewer/internal/handler"
	"github.com/zoyentech/kg-viewer/internal/neo4j"
	"github.com/zoyentech/kg-viewer/pkg/logger"
)

// testLogger is a no-op logger for handler tests.
type testLogger struct{}

func (testLogger) Debugf(string, ...any) {}
func (testLogger) Infof(string, ...any)  {}
func (testLogger) Warnf(string, ...any)  {}
func (testLogger) Errorf(string, ...any) {}
func (testLogger) Fatalf(string, ...any) {}

type fakeGraphLoader struct {
	opts  neo4j.Options
	graph *neo4j.KnowledgeGraph
	err   error
}

func (f *fakeGraphLoader) Fetch(_ context.Context, opts neo4j.Options) (*neo4j.KnowledgeGraph, error) {
	f.opts = opts
	if f.err != nil {
		return nil, f.err
	}
	return f.graph, nil
}

func graphTestRouter(t *testing.T, loader handler.GraphLoader) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewGraphHandler(loader, logger.Logger(testLogger{}))
	r.GET("/v1/graph/kg", h.KG)
	return r
}

func TestGraphKG_ReturnsSnapshot(t *testing.T) {
	loader := &fakeGraphLoader{graph: &neo4j.KnowledgeGraph{
		Nodes:  []neo4j.GraphNode{{ID: "n1", Label: "营养与健康", Type: "Domain"}},
		Links:  []neo4j.GraphLink{{Source: "n1", Target: "n2", Type: "HAS"}},
		Source: "neo4j",
	}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/graph/kg?limit=42&label=Food", nil)
	graphTestRouter(t, loader).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Nodes  []neo4j.GraphNode `json:"nodes"`
			Links  []neo4j.GraphLink `json:"links"`
			Source string            `json:"source"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != 0 || len(body.Data.Nodes) != 1 || body.Data.Source != "neo4j" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
	if loader.opts.Limit != 42 || loader.opts.Label != "Food" {
		t.Fatalf("opts not forwarded: %+v", loader.opts)
	}
}

func TestGraphKG_DefaultLimit(t *testing.T) {
	loader := &fakeGraphLoader{graph: &neo4j.KnowledgeGraph{Source: "neo4j"}}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/graph/kg", nil)
	graphTestRouter(t, loader).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if loader.opts.Limit != 0 { // 0 = client-side default of 300
		t.Fatalf("expected zero limit (client default), got %d", loader.opts.Limit)
	}
}

func TestGraphKG_NotConfigured(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/graph/kg", nil)
	graphTestRouter(t, nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGraphKG_LoaderError(t *testing.T) {
	loader := &fakeGraphLoader{err: errors.New("bolt down")}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/graph/kg", nil)
	graphTestRouter(t, loader).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

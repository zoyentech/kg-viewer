// Package viewer embeds the standalone 3D knowledge-graph viewer and
// serves it at /kg-viewer so the demo page and the /v1/graph/kg API
// share one origin.
package viewer

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed index.html
var indexHTML []byte

// Handler serves the embedded viewer page.
func Handler(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
}

//go:embed snapshot.json
var SnapshotJSON []byte

// SnapshotHandler serves the cached Neo4j graph snapshot so the viewer
// can load offline via ?data=/kg-viewer/snapshot.json.
func SnapshotHandler(c *gin.Context) {
	c.Data(http.StatusOK, "application/json; charset=utf-8", SnapshotJSON)
}

package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
	csvPath := "/Users/mac/zoyen/nutrition-evidence-kg-zh/data/ingredients_with_descriptions.csv"

	f, err := os.Open(csvPath)
	if err != nil {
		fmt.Println("open csv:", err)
		os.Exit(1)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		fmt.Println("read csv:", err)
		os.Exit(1)
	}
	if len(rows) < 2 {
		fmt.Println("csv has no data rows")
		os.Exit(1)
	}

	header := rows[0]
	colIdx := make(map[string]int)
	for i, h := range header {
		if i == 0 {
			h = strings.TrimPrefix(h, "\xef\xbb\xbf")
		}
		colIdx[h] = i
	}
	for _, required := range []string{"ingredient_id", "description"} {
		if _, ok := colIdx[required]; !ok {
			fmt.Println("missing column:", required)
			os.Exit(1)
		}
	}

	uri := "bolt://192.168.31.201:7688"
	user := "neo4j"
	pw := "neo4j-healthdinner-2026"
	db := "neo4j"

	ctx := context.Background()
	drv, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, pw, ""))
	if err != nil {
		fmt.Println("driver:", err)
		os.Exit(1)
	}
	defer drv.Close(ctx)
	if err := drv.VerifyConnectivity(ctx); err != nil {
		fmt.Println("connect:", err)
		os.Exit(1)
	}

	type row struct {
		IID, Desc, Cat, UNII, Role string
		Layer                      int64
	}
	var batch []row

	getCol := func(r []string, name string) string {
		if i, ok := colIdx[name]; ok && i < len(r) {
			return r[i]
		}
		return ""
	}

	for _, r := range rows[1:] {
		layerVal, _ := strconv.ParseInt(getCol(r, "layer"), 10, 64)
		batch = append(batch, row{
			IID:   getCol(r, "ingredient_id"),
			Desc:  getCol(r, "description"),
			Cat:   getCol(r, "category"),
			UNII:  getCol(r, "unii"),
			Role:  getCol(r, "role"),
			Layer: layerVal,
		})
	}

	fmt.Printf("Parsed %d rows. Importing...\n", len(batch))

	const batchSize = 100
	total := 0
	for start := 0; start < len(batch); start += batchSize {
		end := start + batchSize
		if end > len(batch) {
			end = len(batch)
		}
		chunk := batch[start:end]

		paramList := make([]map[string]any, len(chunk))
		for i, r := range chunk {
			paramList[i] = map[string]any{
				"iid":         r.IID,
				"description": r.Desc,
				"category":    r.Cat,
				"unii":        r.UNII,
				"role":        r.Role,
				"layer":       r.Layer,
			}
		}

		cypher := "UNWIND $rows AS row\n" +
			"MATCH (n:\u6210\u5206 {ingredient_id: row.iid})\n" +
			"SET n.description = row.description,\n" +
			"    n.category = row.category,\n" +
			"    n.unii = row.unii,\n" +
			"    n.role = row.role,\n" +
			"    n.layer = row.layer"

		sess := drv.NewSession(ctx, neo4j.SessionConfig{DatabaseName: db})
		_, err := sess.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			_, err := tx.Run(ctx, cypher, map[string]any{"rows": paramList})
			return nil, err
		})
		sess.Close(ctx)
		if err != nil {
			fmt.Printf("batch %d-%d ERROR: %v\n", start, end, err)
			os.Exit(1)
		}
		total += len(chunk)
		fmt.Printf("  imported %d / %d\n", total, len(batch))
	}

	fmt.Printf("Done. %d rows processed.\n", total)
}

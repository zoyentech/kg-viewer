package main

import (
	"context"
	"fmt"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
	uri := "bolt://192.168.31.201:7688"
	user := "neo4j"
	pw := "neo4j-healthdinner-2026"
	db := "neo4j"

	ctx := context.Background()
	drv, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, pw, ""))
	if err != nil {
		fmt.Println("driver err:", err)
		os.Exit(1)
	}
	defer drv.Close(ctx)
	if err := drv.VerifyConnectivity(ctx); err != nil {
		fmt.Println("connect err:", err)
		os.Exit(1)
	}

	sess := drv.NewSession(ctx, neo4j.SessionConfig{DatabaseName: db, AccessMode: neo4j.AccessModeRead})
	defer sess.Close(ctx)

	queries := []struct {
		title, cypher string
	}{
		{"label counts", "CALL db.labels() YIELD label CALL { WITH label MATCH (n) WHERE label IN labels(n) RETURN count(n) AS cnt } RETURN label, cnt ORDER BY cnt DESC"},
		{"sample 成分 node (props)", "MATCH (n:`成分`) RETURN n LIMIT 2"},
		{"has description prop", "MATCH (n:`成分`) WHERE n.description IS NOT NULL RETURN count(n) AS withDesc"},
		{"total 成分 nodes", "MATCH (n:`成分`) RETURN count(n) AS total"},
		{"sample ingredient_id values", "MATCH (n:`成分`) RETURN n.ingredient_id AS iid, n.canonical_name AS cn LIMIT 5"},
	}

	for _, q := range queries {
		fmt.Printf("\n=== %s ===\n", q.title)
		res, err := sess.Run(ctx, q.cypher, nil)
		if err != nil {
			fmt.Println("  ERROR:", err)
			continue
		}
		for res.Next(ctx) {
			rec := res.Record()
			for i, k := range rec.Keys {
				fmt.Printf("  %s = %v\n", k, rec.Values[i])
			}
			fmt.Println("  ---")
		}
		if err := res.Err(); err != nil {
			fmt.Println("  READ ERR:", err)
		}
	}
}

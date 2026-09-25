// Nutrition evidence dataset v3.
// Run against the standalone Community instance on Bolt 7689 after the
// existing graph has been backed up and cleared. The v3 graph has four
// visible labels: Product, Ingredient, Evidence, and Outcome.

CREATE CONSTRAINT v3_product_id IF NOT EXISTS
FOR (n:Product) REQUIRE n.product_id IS UNIQUE;
CREATE CONSTRAINT v3_ingredient_id IF NOT EXISTS
FOR (n:Ingredient) REQUIRE n.ingredient_id IS UNIQUE;
CREATE CONSTRAINT v3_evidence_id IF NOT EXISTS
FOR (n:Evidence) REQUIRE n.evidence_id IS UNIQUE;
CREATE CONSTRAINT v3_outcome_id IF NOT EXISTS
FOR (n:Outcome) REQUIRE n.outcome_id IS UNIQUE;

LOAD CSV WITH HEADERS FROM 'file:///products.csv' AS row
CALL {
  WITH row
  MERGE (n:Product {product_id: row.product_id})
  SET n.name_en = row.name_en,
      n.name_zh = row.name_zh,
      n.brand = row.brand,
      n.description = row.description,
      n.layer = row.layer,
      n.data_version = 'v3'
} IN TRANSACTIONS OF 1000 ROWS;

LOAD CSV WITH HEADERS FROM 'file:///ingredients.csv' AS row
CALL {
  WITH row
  MERGE (n:Ingredient {ingredient_id: row.ingredient_id})
  SET n.name_en = row.ingredient_name_en,
      n.name_zh = row.ingredient_name_ch,
      n.canonical_name = row.ingredient_name_en,
      n.description = row.description,
      n.layer = row.layer,
      n.data_quality = 'complete',
      n.data_version = 'v3'
} IN TRANSACTIONS OF 1000 ROWS;

LOAD CSV WITH HEADERS FROM 'file:///evidence.csv' AS row
CALL {
  WITH row
  MERGE (n:Evidence {evidence_id: row.evidence_id})
  SET n.title = row.title,
      n.title_en = row.title_en,
      n.title_zh = row.title_zh,
      n.summary = row.summary,
      n.conclusion = row.conclusion,
      n.evidence_level = row.evidence_level,
      n.source_url = row.source_url,
      n.ingredient_name = row.ingredient_name,
      n.outcome_name = row.outcome_name,
      n.layer = row.layer,
      n.data_version = 'v3'
} IN TRANSACTIONS OF 500 ROWS;

LOAD CSV WITH HEADERS FROM 'file:///outcomes.csv' AS row
CALL {
  WITH row
  MERGE (n:Outcome {outcome_id: row.outcome_id})
  SET n.name_en = row.name_en,
      n.name_zh = row.name_zh,
      n.description = row.description,
      n.description_zh = row.description_zh,
      n.layer = row.layer,
      n.data_version = 'v3'
} IN TRANSACTIONS OF 500 ROWS;

LOAD CSV WITH HEADERS FROM 'file:///product_ingredients.csv' AS row
CALL {
  WITH row
  MATCH (p:Product {product_id: row.product_id})
  MATCH (i:Ingredient {ingredient_id: row.ingredient_id})
  MERGE (p)-[r:DECLARES_INGREDIENT]->(i)
  SET r.ingredient_name = row.ingredient_name,
      r.role = row.role,
      r.amount = row.amount
} IN TRANSACTIONS OF 2000 ROWS;

LOAD CSV WITH HEADERS FROM 'file:///ingredient_evidence.csv' AS row
CALL {
  WITH row
  MATCH (i:Ingredient {ingredient_id: row.ingredient_id})
  MATCH (e:Evidence {evidence_id: row.evidence_id})
  MERGE (i)-[:EVIDENCE_FOR]->(e)
} IN TRANSACTIONS OF 1000 ROWS;

LOAD CSV WITH HEADERS FROM 'file:///evidence_outcome.csv' AS row
CALL {
  WITH row
  MATCH (e:Evidence {evidence_id: row.evidence_id})
  MATCH (o:Outcome {outcome_id: row.outcome_id})
  MERGE (e)-[:SUPPORTS_OUTCOME]->(o)
} IN TRANSACTIONS OF 1000 ROWS;

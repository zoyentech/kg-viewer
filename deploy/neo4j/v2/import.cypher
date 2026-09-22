// Nutrition evidence dataset v2.
// Run against the standalone Community instance on 7689.
// CSV files are mounted under server.directories.import.

CREATE CONSTRAINT v2_product_id IF NOT EXISTS
FOR (n:Product) REQUIRE n.product_id IS UNIQUE;
CREATE CONSTRAINT v2_ingredient_id IF NOT EXISTS
FOR (n:Ingredient) REQUIRE n.ingredient_id IS UNIQUE;
CREATE CONSTRAINT v2_evidence_id IF NOT EXISTS
FOR (n:Evidence) REQUIRE n.evidence_id IS UNIQUE;
CREATE CONSTRAINT v2_topic_id IF NOT EXISTS
FOR (n:HealthTopic) REQUIRE n.topic_id IS UNIQUE;
CREATE CONSTRAINT v2_context_id IF NOT EXISTS
FOR (n:EvidenceContext) REQUIRE n.context_id IS UNIQUE;

LOAD CSV WITH HEADERS FROM 'file:///products.csv' AS row
CALL {
  WITH row
  MERGE (n:Product {product_id: row.product_id})
  SET n.name_en = row.name_en,
      n.name_zh = row.name_zh,
      n.brand = row.brand,
      n.description = row.description,
      n.layer = row.layer,
      n.data_version = 'v2'
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
      n.data_version = 'v2'
} IN TRANSACTIONS OF 1000 ROWS;

LOAD CSV WITH HEADERS FROM 'file:///evidence.csv' AS row
CALL {
  WITH row
  MERGE (n:Evidence {evidence_id: row.evidence_id})
  SET n.title = row.title,
      n.title_zh = row.title,
      n.pmid = row.pmid,
      n.doi = row.doi,
      n.year = row.year,
      n.study_type = row.study_type,
      n.design_tier = row.design_tier,
      n.direction_hint = row.direction_hint,
      n.direction_method = row.direction_method,
      n.validation_status = row.validation_status,
      n.abstract_excerpt = row.abstract_excerpt,
      n.source_url = row.source_url,
      n.layer = row.layer,
      n.data_version = 'v2'
} IN TRANSACTIONS OF 500 ROWS;

LOAD CSV WITH HEADERS FROM 'file:///health_topics.csv' AS row
CALL {
  WITH row
  MERGE (n:HealthTopic {topic_id: row.topic_id})
  SET n.name = row.name,
      n.ontology_id = row.ontology_id,
      n.layer = row.layer,
      n.data_version = 'v2'
} IN TRANSACTIONS OF 500 ROWS;

// Preserve orphan evidence references as explicit, reviewable placeholders.
// They are completed by the ingredients.csv import when a matching row exists.
LOAD CSV WITH HEADERS FROM 'file:///evidence_links.csv' AS row
CALL {
  WITH row
  MERGE (n:Ingredient {ingredient_id: row.ingredient_id})
  ON CREATE SET n.name_en = '[missing ingredient] ' + row.ingredient_id,
                n.name_zh = '[缺少成分记录] ' + row.ingredient_id,
                n.data_quality = 'missing_from_ingredients_csv',
                n.data_version = 'v2'
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

LOAD CSV WITH HEADERS FROM 'file:///evidence_links.csv' AS row
CALL {
  WITH row
  MATCH (i:Ingredient {ingredient_id: row.ingredient_id})
  MATCH (e:Evidence {evidence_id: row.evidence_id})
  MATCH (t:HealthTopic {topic_id: row.topic_id})
  MERGE (c:EvidenceContext {context_id: row.context_id})
  SET c.search_query = row.search_query,
      c.retrieved_on = row.retrieved_on,
      c.layer = 'context',
      c.data_version = 'v2'
  MERGE (i)-[:HAS_EVIDENCE_CONTEXT]->(c)
  MERGE (c)-[:CITES]->(e)
  MERGE (c)-[:ABOUT_TOPIC]->(t)
} IN TRANSACTIONS OF 500 ROWS;

LOAD CSV WITH HEADERS FROM 'file:///ingredient_topics.csv' AS row
CALL {
  WITH row
  MATCH (i:Ingredient {ingredient_id: row.ingredient_id})
  MATCH (t:HealthTopic {topic_id: row.topic_id})
  MERGE (i)-[r:INVESTIGATED_FOR]->(t)
  SET r.selection_method = row.selection_method
} IN TRANSACTIONS OF 1000 ROWS;

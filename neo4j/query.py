#

# CALL apoc.load.csv("/data/dataset/oi-warehouse/aggregation/what=graph_vertices/source=tranco/year=2025/month=07/day=11/*.csv") YIELD map AS row MERGE (:Domain {id: row.id});

#CALL apoc.load.csv('/data/dataset/oi-warehouse/aggregation/what=graph_edges/source=tranco/year=2025/month=07/day=11/*.csv') YIELD map AS row
#MATCH (s:Domain {id: row.src}), (t:Domain {id: row.dst})
#FOREACH(_ IN CASE WHEN row.edge_type='rua' THEN [1] ELSE [] END |
#  CREATE (s)-[:RUA]->(t)
#)
#FOREACH(_ IN CASE WHEN row.edge_type='ruf' THEN [1] ELSE [] END |
#  CREATE (s)-[:RUF]->(t)
#);

# worked!
#CREATE INDEX vertices_id_index FOR (n:Vertices) ON (n.id);
:auto USING PERIODIC COMMIT 5000
LOAD CSV WITH HEADERS FROM "file:///vertices.csv" AS row
MERGE (:Vertices {id: row.id});


:auto USING PERIODIC COMMIT 5000
LOAD CSV WITH HEADERS FROM "file:///rua_edges.csv" AS row
MATCH (source:Vertices {id: row.src})
MATCH (destination:Vertices {id: row.dst})
MERGE (source)-[:RUA]->(destination);

:auto USING PERIODIC COMMIT 5000
LOAD CSV WITH HEADERS FROM "file:///ruf_edges.csv" AS row
MATCH (source:Vertices {id: row.src})
MATCH (destination:Vertices {id: row.dst})
MERGE (source)-[:RUF]->(destination);

MATCH (n:Vertices)-[r]->(m:Vertices) 
RETURN n, r, m 
LIMIT 50;
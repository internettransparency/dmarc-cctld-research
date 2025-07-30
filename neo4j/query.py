# create vertices
:auto USING PERIODIC COMMIT 5000
LOAD CSV WITH HEADERS FROM "file:///vertices.csv" AS row
MERGE (:Vertices {id: row.id});

# rua relationships
:auto USING PERIODIC COMMIT 5000
LOAD CSV WITH HEADERS FROM "file:///rua_edges.csv" AS row
MATCH (source:Vertices {id: row.src})
MATCH (destination:Vertices {id: row.dst})
MERGE (source)-[:RUA]->(destination);

# ruf relationships
:auto USING PERIODIC COMMIT 5000
LOAD CSV WITH HEADERS FROM "file:///ruf_edges.csv" AS row
MATCH (source:Vertices {id: row.src})
MATCH (destination:Vertices {id: row.dst})
MERGE (source)-[:RUF]->(destination);

# visualize the graph
MATCH (n:Vertices)-[r]->(m:Vertices) 
RETURN n, r, m 
LIMIT 50;

# delete all relationships
MATCH ()-[r]-()
DELETE r

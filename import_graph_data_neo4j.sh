#!/bin/bash

rm -rf neo4j/import/*

echo "id,is_gov" > neo4j/import/vertices.csv
echo "src,dst,edge_type" > neo4j/import/edges.csv

cat data/oi-warehouse/aggregation/what=graph_vertices/source=tranco/year=2025/month=07/day=11/*.csv >> neo4j/import/vertices.csv
cat data/oi-warehouse/aggregation/what=graph_edges/source=tranco/year=2025/month=07/day=11/*.csv >> neo4j/import/edges.csv

cd neo4j/import/ || exit 1
# no need to separate files...
# Extract header and rua edges
#(head -n1 edges.csv; grep ",rua$" edges.csv) > rua_edges.csv

# Extract header and ruf edges
#(head -n1 edges.csv; grep ",ruf$" edges.csv) > ruf_edges.csv

#rm -f edges.csv

echo "Data prepared for Neo4j import."

cd - || exit 1

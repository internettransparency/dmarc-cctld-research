#!/bin/bash


podman-compose -p neo4jimport run --rm --no-deps \
	neo4j \
	neo4j-admin database import full neo4j \
	--overwrite-destination=true \
	--array-delimiter=";" \
	--nodes=Vertices=/var/lib/neo4j/import/vertices.csv \
	--relationships=/var/lib/neo4j/import/edges.csv \
	--verbose

podman pod stop neo4jimport
podman pod rm neo4jimport


#!/bin/bash

PODMAN_USERNS=keep-id podman-compose -p neo4jimport run --rm --no-deps neo4j \
	neo4j-admin database import full neo4j  \
	--overwrite-destination=true  \
	--nodes=Vertices=/var/lib/neo4j/import/vertices.csv \
	--relationships=/var/lib/neo4j/import/edges.csv

PODMAN_USERNS=keep-id podman-compose up -d

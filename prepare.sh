#!/bin/bash

mkdir -p neo4j/{conf,data,import,logs,plugins}
wget -O neo4j/plugins https://github.com/neo4j-contrib/neo4j-apoc-procedures/releases/download/4.4.0.37/apoc-4.4.0.37-all.jar

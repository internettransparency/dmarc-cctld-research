#!/bin/bash

bucket_name="luvizottocesarg-tmp"
alias_name="tmp_rw"

declare -a remote_files=(
    "oi-warehouse/aggregation/what=graph_edges/"
    "oi-warehouse/aggregation/what=graph_vertices/"
    "oi-warehouse/testing/source=tranco/cctld_dmarc/"
)

for file in "${remote_files[@]}"; do
    echo "Downloading ${file}..."
    filepath=$(mc ls -r --json --dp -no-color "${alias_name}/${bucket_name}/${file}" | jq 'select(.key | endswith(".parquet")) | .key ' | sort | tail -n1 | tr -d '"')
    directory=$(dirname "${filepath}")
    rm -rf  "data/${file}"
    mkdir -p "data/${file}${directory}"
    mc cp --recursive --no-color --dp "${alias_name}/${bucket_name}/${file}" "data/${file}"
done

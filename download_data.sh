#!/bin/bash

bucket_name="luvizottocesarg-tmp"
alias_name="tmp_rw"

declare -a remote_files=(
    # for RQ1
    "oi-warehouse/aggregation/what=rua_addresses/"
    "oi-warehouse/aggregation/what=ruf_addresses/"
    "oi-warehouse/aggregation/what=graph_edges/"
    "oi-warehouse/aggregation/what=graph_vertices/"
)

for file in "${remote_files[@]}"; do
    echo "Downloading ${file}..."
    filepath=$(mc ls -r --json --dp -no-color "${alias_name}/${bucket_name}/${file}" | jq 'select(.key | endswith(".parquet")) | .key ' | sort | tail -n1 | tr -d '"')
    directory=$(dirname "${filepath}")
    mkdir -p "data/${file}${directory}"
    mc cp --recursive --no-color --dp "${alias_name}/${bucket_name}/${file}" "data/${file}"
done

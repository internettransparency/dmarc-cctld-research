#!/bin/bash

bucket_name="luvizottocesarg-tmp"
alias_name="tmp_rw"

declare -a remote_files=(
    "oi-warehouse/source=ct_logs/graph_edges/"
    "oi-warehouse/source=ct_logs/graph_vertices/"
    "oi-warehouse/source=ct_logs/cctld_dmarc/"
    "oi-warehouse/source=ct_logs/countries_counts/"
    "oi-warehouse/source=new_meas/rua_ruf_country/"
)

for file in "${remote_files[@]}"; do
    echo "Downloading ${file}..."
    filepath=$(mc ls -r --json --dp -no-color "${alias_name}/${bucket_name}/${file}" | jq 'select(.key | endswith(".parquet")) | .key ' | sort | tail -n1 | tr -d '"')
    directory=$(dirname "${filepath}")
    rm -rf  "data/${file}"
    mkdir -p "data/${file}${directory}"
    mc cp --recursive --no-color --dp "${alias_name}/${bucket_name}/${file}" "data/${file}"
done


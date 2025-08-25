from datetime import datetime
import pandas as pd


def main():
    # Load the Parquet files
    start_date = datetime(2025, 7, 11)
    what = "graph_vertices"
    parquet_data_path = "data/oi-warehouse/testing/aggregation/what={what}/source=tranco/year={start_date.year}/month={start_date.month:02d}/day={start_date.day:02d}/"
    print(parquet_data_path.format(what=what, start_date=start_date))
    vertices_pdf = pd.read_parquet(parquet_data_path.format(what=what, start_date=start_date))
    what = "graph_edges"
    print(parquet_data_path.format(what=what, start_date=start_date))
    edges_pdf = pd.read_parquet(parquet_data_path.format(what=what, start_date=start_date))

    # Save the DataFrames as CSV files
    vertices_pdf.to_csv("neo4j/import/vertices.csv", index=False)
    edges_pdf.to_csv("neo4j/import/edges.csv", index=False)

    vertices_pdf = pd.read_csv("neo4j/import/vertices.csv")
    edges_pdf = pd.read_csv("neo4j/import/edges.csv")
    rua_edges = edges_pdf[edges_pdf['edge_type'] == 'rua']
    ruf_edges = edges_pdf[edges_pdf['edge_type'] == 'ruf']
    print(f"Data prepared for Neo4j import. Vertices: {len(vertices_pdf)}. Edges: {len(edges_pdf)}. RUA Edges: {len(rua_edges)}. RUF Edges: {len(ruf_edges)}.")


if __name__ == "__main__":
    main()

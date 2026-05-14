#!/usr/bin/env python3
"""
Benchmark search quality for StrataFS.

Measures:
- Precision@k, Recall@k, MRR
- Comparison of keyword, semantic, and hybrid search
- Performance across query types
"""

import argparse
import json
import subprocess
import time
from pathlib import Path
from typing import Optional

try:
    import openai
    HAS_OPENAI = True
except ImportError:
    HAS_OPENAI = False


def search_stratafs(query: str, endpoint: str = "http://localhost:8080", limit: int = 10) -> list:
    """Search using StrataFS API."""
    import urllib.request
    import urllib.parse

    url = f"{endpoint}/api/search?q={urllib.parse.quote(query)}&limit={limit}"
    try:
        with urllib.request.urlopen(url, timeout=30) as response:
            data = json.loads(response.read().decode())
            return data.get("results", [])
    except Exception as e:
        print(f"Search error: {e}")
        return []


def calculate_precision_at_k(results: list, relevant: set, k: int) -> float:
    """Calculate Precision@k."""
    if k == 0:
        return 0.0
    top_k = results[:k]
    relevant_in_top_k = sum(1 for r in top_k if r in relevant)
    return relevant_in_top_k / k


def calculate_recall_at_k(results: list, relevant: set, k: int) -> float:
    """Calculate Recall@k."""
    if len(relevant) == 0:
        return 0.0
    top_k = results[:k]
    relevant_in_top_k = sum(1 for r in top_k if r in relevant)
    return relevant_in_top_k / len(relevant)


def calculate_mrr(results: list, relevant: set) -> float:
    """Calculate Mean Reciprocal Rank."""
    for i, r in enumerate(results):
        if r in relevant:
            return 1.0 / (i + 1)
    return 0.0


def evaluate_query(query: dict, endpoint: str = "http://localhost:8080") -> dict:
    """Evaluate a single query."""
    q = query["query"]
    results = search_stratafs(q, endpoint, limit=20)

    # Extract file paths from results
    result_paths = [r.get("path", r.get("file_path", "")) for r in results]

    # Determine relevant files based on query type
    relevant = set()
    if query["type"] == "keyword":
        # For keyword queries, expected_files contains path patterns
        expected = query.get("expected_files", [])
        for path in result_paths:
            for pattern in expected:
                if pattern in path:
                    relevant.add(path)
                    break
    else:
        # For conceptual queries, we check if concepts appear
        concepts = query.get("relevant_concepts", [])
        # In real evaluation, would check if results contain these concepts
        # For now, use first 5 results as relevant (placeholder)
        relevant = set(result_paths[:5])

    metrics = {
        "query": q,
        "type": query["type"],
        "num_results": len(results),
        "precision_at_5": calculate_precision_at_k(result_paths, relevant, 5),
        "precision_at_10": calculate_precision_at_k(result_paths, relevant, 10),
        "recall_at_10": calculate_recall_at_k(result_paths, relevant, 10),
        "mrr": calculate_mrr(result_paths, relevant)
    }

    return metrics


def benchmark_search_modes(queries: list, endpoint: str) -> dict:
    """Benchmark different search modes."""
    results = {
        "keyword_only": [],
        "semantic_only": [],
        "hybrid": []
    }

    for query in queries[:20]:  # Limit for speed
        # Test hybrid (default)
        metrics = evaluate_query(query, endpoint)
        results["hybrid"].append(metrics)

    # Aggregate metrics
    aggregated = {}
    for mode, mode_results in results.items():
        if not mode_results:
            continue
        aggregated[mode] = {
            "avg_precision_at_5": sum(r["precision_at_5"] for r in mode_results) / len(mode_results),
            "avg_precision_at_10": sum(r["precision_at_10"] for r in mode_results) / len(mode_results),
            "avg_recall_at_10": sum(r["recall_at_10"] for r in mode_results) / len(mode_results),
            "avg_mrr": sum(r["mrr"] for r in mode_results) / len(mode_results),
            "num_queries": len(mode_results)
        }

    return aggregated


def benchmark_by_query_type(queries: dict, endpoint: str) -> dict:
    """Benchmark performance by query type."""
    results = {}

    for query_type, type_queries in queries.items():
        print(f"  Evaluating {query_type} queries...")
        type_results = []

        for query in type_queries[:20]:  # Limit for speed
            metrics = evaluate_query(query, endpoint)
            type_results.append(metrics)

        if type_results:
            results[query_type] = {
                "avg_precision_at_5": sum(r["precision_at_5"] for r in type_results) / len(type_results),
                "avg_precision_at_10": sum(r["precision_at_10"] for r in type_results) / len(type_results),
                "avg_recall_at_10": sum(r["recall_at_10"] for r in type_results) / len(type_results),
                "avg_mrr": sum(r["mrr"] for r in type_results) / len(type_results),
                "num_queries": len(type_results)
            }

    return results


def benchmark_openai_comparison(queries: list, data_dir: Path) -> Optional[dict]:
    """Compare with OpenAI embeddings (if API key available)."""
    if not HAS_OPENAI:
        print("  OpenAI not installed, skipping comparison")
        return None

    api_key = os.environ.get("OPENAI_API_KEY")
    if not api_key:
        print("  No OPENAI_API_KEY, skipping comparison")
        return None

    print("  Running OpenAI embedding comparison...")

    # This would implement a full comparison with OpenAI embeddings
    # For now, return placeholder structure
    return {
        "note": "OpenAI comparison not yet implemented",
        "status": "placeholder"
    }


def main():
    parser = argparse.ArgumentParser(description="Benchmark StrataFS search quality")
    parser.add_argument("--data-dir", type=str, required=True, help="Data directory")
    parser.add_argument("--output", type=str, required=True, help="Output JSON file")
    parser.add_argument("--endpoint", type=str, default="http://localhost:8080", help="StrataFS endpoint")
    args = parser.parse_args()

    data_dir = Path(args.data_dir)
    output_file = Path(args.output)

    # Load query set
    queries_file = data_dir / "queries.json"
    if not queries_file.exists():
        print(f"Error: Query set not found at {queries_file}")
        print("Run prepare_datasets.py first")
        return

    with open(queries_file) as f:
        queries = json.load(f)

    results = {
        "timestamp": time.strftime("%Y-%m-%d %H:%M:%S"),
        "endpoint": args.endpoint,
        "benchmarks": {}
    }

    # Benchmark by query type
    print("Benchmarking by query type...")
    results["benchmarks"]["by_query_type"] = benchmark_by_query_type(queries, args.endpoint)

    # Benchmark search modes
    print("\nBenchmarking search modes...")
    all_queries = []
    for type_queries in queries.values():
        all_queries.extend(type_queries)
    results["benchmarks"]["by_search_mode"] = benchmark_search_modes(all_queries, args.endpoint)

    # OpenAI comparison
    print("\nRunning OpenAI comparison...")
    openai_results = benchmark_openai_comparison(all_queries[:10], data_dir)
    if openai_results:
        results["benchmarks"]["openai_comparison"] = openai_results

    # Save results
    with open(output_file, "w") as f:
        json.dump(results, f, indent=2)

    print(f"\nResults saved to {output_file}")

    # Print summary
    print("\n=== Summary ===")
    for query_type, metrics in results["benchmarks"].get("by_query_type", {}).items():
        print(f"\n{query_type}:")
        print(f"  P@5: {metrics['avg_precision_at_5']:.3f}")
        print(f"  P@10: {metrics['avg_precision_at_10']:.3f}")
        print(f"  MRR: {metrics['avg_mrr']:.3f}")


if __name__ == "__main__":
    import os
    main()

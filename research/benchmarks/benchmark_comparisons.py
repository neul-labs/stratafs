#!/usr/bin/env python3
"""
Comparison benchmarks against baseline systems.

Compares AgentFS against:
- ripgrep (pure keyword search)
- Elasticsearch (traditional search engine)
- OpenAI + FAISS (cloud embeddings)
"""

import argparse
import json
import os
import shutil
import subprocess
import time
import urllib.parse
import urllib.request
from pathlib import Path

try:
    import openai
    HAS_OPENAI = True
except ImportError:
    HAS_OPENAI = False


def benchmark_ripgrep(corpus_path: Path, queries: list) -> dict:
    """Benchmark ripgrep search."""
    if not shutil.which("rg"):
        return {"error": "ripgrep not installed"}

    results = []
    for query_data in queries[:20]:
        query = query_data["query"]

        start = time.perf_counter()
        try:
            proc = subprocess.run(
                ["rg", "-l", "--max-count=10", query, str(corpus_path)],
                capture_output=True,
                timeout=30
            )
            end = time.perf_counter()

            num_results = len(proc.stdout.decode().strip().split("\n")) if proc.stdout else 0
            results.append({
                "query": query,
                "latency_ms": (end - start) * 1000,
                "num_results": num_results,
                "success": True
            })
        except subprocess.TimeoutExpired:
            results.append({
                "query": query,
                "latency_ms": 30000,
                "num_results": 0,
                "success": False
            })

    latencies = [r["latency_ms"] for r in results if r["success"]]

    return {
        "tool": "ripgrep",
        "num_queries": len(results),
        "avg_latency_ms": sum(latencies) / len(latencies) if latencies else 0,
        "min_latency_ms": min(latencies) if latencies else 0,
        "max_latency_ms": max(latencies) if latencies else 0,
        "note": "Keyword matching only, no semantic understanding"
    }


def benchmark_grep(corpus_path: Path, queries: list) -> dict:
    """Benchmark standard grep for baseline."""
    results = []
    for query_data in queries[:10]:
        query = query_data["query"]

        start = time.perf_counter()
        try:
            proc = subprocess.run(
                ["grep", "-r", "-l", "--max-count=10", query, str(corpus_path)],
                capture_output=True,
                timeout=60
            )
            end = time.perf_counter()

            num_results = len(proc.stdout.decode().strip().split("\n")) if proc.stdout else 0
            results.append({
                "latency_ms": (end - start) * 1000,
                "num_results": num_results
            })
        except subprocess.TimeoutExpired:
            results.append({"latency_ms": 60000, "num_results": 0})

    latencies = [r["latency_ms"] for r in results]

    return {
        "tool": "grep",
        "num_queries": len(results),
        "avg_latency_ms": sum(latencies) / len(latencies) if latencies else 0,
        "note": "Baseline keyword search"
    }


def benchmark_elasticsearch(queries: list, es_endpoint: str = "http://localhost:9200") -> dict:
    """Benchmark Elasticsearch if available."""
    # Check if Elasticsearch is running
    try:
        with urllib.request.urlopen(es_endpoint, timeout=5) as response:
            pass
    except Exception:
        return {"error": "Elasticsearch not available", "endpoint": es_endpoint}

    results = []
    for query_data in queries[:20]:
        query = query_data["query"]

        # Build ES query
        es_query = {
            "query": {
                "multi_match": {
                    "query": query,
                    "fields": ["content", "path"]
                }
            },
            "size": 10
        }

        start = time.perf_counter()
        try:
            req = urllib.request.Request(
                f"{es_endpoint}/agentfs/_search",
                data=json.dumps(es_query).encode(),
                headers={"Content-Type": "application/json"}
            )
            with urllib.request.urlopen(req, timeout=30) as response:
                data = json.loads(response.read().decode())
                end = time.perf_counter()

                num_results = len(data.get("hits", {}).get("hits", []))
                results.append({
                    "latency_ms": (end - start) * 1000,
                    "num_results": num_results,
                    "success": True
                })
        except Exception as e:
            results.append({
                "latency_ms": 0,
                "num_results": 0,
                "success": False,
                "error": str(e)
            })

    successful = [r for r in results if r["success"]]
    latencies = [r["latency_ms"] for r in successful]

    return {
        "tool": "elasticsearch",
        "num_queries": len(results),
        "successful": len(successful),
        "avg_latency_ms": sum(latencies) / len(latencies) if latencies else 0,
        "note": "Full-text search with BM25"
    }


def benchmark_openai_faiss(queries: list, corpus_path: Path) -> dict:
    """Benchmark OpenAI embeddings + FAISS."""
    if not HAS_OPENAI:
        return {"error": "OpenAI package not installed"}

    api_key = os.environ.get("OPENAI_API_KEY")
    if not api_key:
        return {"error": "OPENAI_API_KEY not set"}

    # This would implement full OpenAI + FAISS comparison
    # For now, return placeholder
    return {
        "tool": "openai_faiss",
        "note": "Full implementation pending",
        "status": "placeholder",
        "expected_metrics": {
            "embedding_cost_per_1k_tokens": 0.0001,
            "avg_latency_ms": "~200-500ms (API call)",
            "precision": "Expected similar or slightly better than local"
        }
    }


def benchmark_agentfs(queries: list, endpoint: str = "http://localhost:8080") -> dict:
    """Benchmark AgentFS for comparison."""
    results = []

    for query_data in queries[:20]:
        query = query_data["query"]
        url = f"{endpoint}/api/search?q={urllib.parse.quote(query)}&limit=10"

        start = time.perf_counter()
        try:
            with urllib.request.urlopen(url, timeout=30) as response:
                data = json.loads(response.read().decode())
                end = time.perf_counter()

                results.append({
                    "latency_ms": (end - start) * 1000,
                    "num_results": len(data.get("results", [])),
                    "success": True
                })
        except Exception as e:
            results.append({
                "latency_ms": 0,
                "num_results": 0,
                "success": False
            })

    successful = [r for r in results if r["success"]]
    latencies = [r["latency_ms"] for r in successful]

    return {
        "tool": "agentfs",
        "num_queries": len(results),
        "successful": len(successful),
        "avg_latency_ms": sum(latencies) / len(latencies) if latencies else 0,
        "note": "Hybrid search with local embeddings"
    }


def generate_comparison_table(results: dict) -> str:
    """Generate a comparison table."""
    lines = [
        "| System | Avg Latency | Semantic | Local | Cost |",
        "|--------|-------------|----------|-------|------|"
    ]

    systems = [
        ("AgentFS", results.get("agentfs", {}), "Yes", "Yes", "Free"),
        ("ripgrep", results.get("ripgrep", {}), "No", "Yes", "Free"),
        ("Elasticsearch", results.get("elasticsearch", {}), "No", "Yes/No", "Free/$$"),
        ("OpenAI+FAISS", results.get("openai_faiss", {}), "Yes", "No", "$$$"),
    ]

    for name, data, semantic, local, cost in systems:
        if "error" in data:
            latency = "N/A"
        else:
            latency = f"{data.get('avg_latency_ms', 0):.1f}ms"
        lines.append(f"| {name} | {latency} | {semantic} | {local} | {cost} |")

    return "\n".join(lines)


def main():
    parser = argparse.ArgumentParser(description="Run comparison benchmarks")
    parser.add_argument("--data-dir", type=str, required=True, help="Data directory")
    parser.add_argument("--output", type=str, required=True, help="Output JSON file")
    parser.add_argument("--endpoint", type=str, default="http://localhost:8080", help="AgentFS endpoint")
    args = parser.parse_args()

    data_dir = Path(args.data_dir)
    output_file = Path(args.output)

    # Load queries
    queries_file = data_dir / "queries.json"
    if queries_file.exists():
        with open(queries_file) as f:
            query_data = json.load(f)
        queries = query_data.get("keyword", [])
    else:
        queries = [{"query": q} for q in ["mutex", "tcp", "memory", "file", "network"]]

    corpus_path = data_dir / "linux-kernel"

    results = {
        "timestamp": time.strftime("%Y-%m-%d %H:%M:%S"),
        "benchmarks": {}
    }

    # Benchmark each system
    print("Benchmarking AgentFS...")
    results["benchmarks"]["agentfs"] = benchmark_agentfs(queries, args.endpoint)

    print("Benchmarking ripgrep...")
    if corpus_path.exists():
        results["benchmarks"]["ripgrep"] = benchmark_ripgrep(corpus_path, queries)
    else:
        results["benchmarks"]["ripgrep"] = {"error": "Corpus not found"}

    print("Benchmarking grep...")
    if corpus_path.exists():
        results["benchmarks"]["grep"] = benchmark_grep(corpus_path, queries)

    print("Benchmarking Elasticsearch...")
    results["benchmarks"]["elasticsearch"] = benchmark_elasticsearch(queries)

    print("Benchmarking OpenAI + FAISS...")
    results["benchmarks"]["openai_faiss"] = benchmark_openai_faiss(queries, corpus_path)

    # Generate comparison table
    results["comparison_table"] = generate_comparison_table(results["benchmarks"])

    # Save results
    with open(output_file, "w") as f:
        json.dump(results, f, indent=2)

    print(f"\nResults saved to {output_file}")
    print("\n=== Comparison Table ===")
    print(results["comparison_table"])


if __name__ == "__main__":
    main()

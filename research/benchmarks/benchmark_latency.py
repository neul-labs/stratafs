#!/usr/bin/env python3
"""
Benchmark search latency for AgentFS.

Measures:
- P50, P95, P99 latencies
- Latency vs corpus size
- Concurrent query performance
"""

import argparse
import json
import statistics
import time
import urllib.parse
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path


def search_with_timing(query: str, endpoint: str = "http://localhost:8080") -> dict:
    """Execute search and return timing."""
    url = f"{endpoint}/api/search?q={urllib.parse.quote(query)}&limit=10"

    start = time.perf_counter()
    try:
        with urllib.request.urlopen(url, timeout=30) as response:
            data = json.loads(response.read().decode())
            end = time.perf_counter()
            return {
                "latency_ms": (end - start) * 1000,
                "num_results": len(data.get("results", [])),
                "success": True
            }
    except Exception as e:
        end = time.perf_counter()
        return {
            "latency_ms": (end - start) * 1000,
            "num_results": 0,
            "success": False,
            "error": str(e)
        }


def calculate_percentiles(latencies: list) -> dict:
    """Calculate latency percentiles."""
    if not latencies:
        return {"p50": 0, "p95": 0, "p99": 0, "mean": 0, "min": 0, "max": 0}

    sorted_latencies = sorted(latencies)
    n = len(sorted_latencies)

    return {
        "p50": sorted_latencies[int(n * 0.50)],
        "p95": sorted_latencies[int(n * 0.95)] if n > 20 else sorted_latencies[-1],
        "p99": sorted_latencies[int(n * 0.99)] if n > 100 else sorted_latencies[-1],
        "mean": statistics.mean(latencies),
        "min": min(latencies),
        "max": max(latencies),
        "std": statistics.stdev(latencies) if len(latencies) > 1 else 0
    }


def benchmark_single_query_latency(queries: list, endpoint: str, num_iterations: int = 100) -> dict:
    """Benchmark single query latency."""
    latencies = []

    for i in range(num_iterations):
        query = queries[i % len(queries)]
        result = search_with_timing(query, endpoint)
        if result["success"]:
            latencies.append(result["latency_ms"])

    return {
        "num_queries": len(latencies),
        "percentiles": calculate_percentiles(latencies)
    }


def benchmark_concurrent_queries(queries: list, endpoint: str, concurrency: int, num_queries: int = 100) -> dict:
    """Benchmark concurrent query performance."""
    latencies = []
    errors = 0

    with ThreadPoolExecutor(max_workers=concurrency) as executor:
        futures = []
        for i in range(num_queries):
            query = queries[i % len(queries)]
            futures.append(executor.submit(search_with_timing, query, endpoint))

        for future in as_completed(futures):
            result = future.result()
            if result["success"]:
                latencies.append(result["latency_ms"])
            else:
                errors += 1

    return {
        "concurrency": concurrency,
        "num_queries": num_queries,
        "successful": len(latencies),
        "errors": errors,
        "percentiles": calculate_percentiles(latencies),
        "throughput_qps": len(latencies) / (sum(latencies) / 1000) if latencies else 0
    }


def benchmark_query_complexity(endpoint: str) -> dict:
    """Benchmark latency vs query complexity."""
    results = {}

    # Short keyword queries
    short_queries = ["mutex", "tcp", "alloc", "print", "file"]
    latencies = []
    for q in short_queries * 20:
        result = search_with_timing(q, endpoint)
        if result["success"]:
            latencies.append(result["latency_ms"])
    results["short_keyword"] = calculate_percentiles(latencies)

    # Long conceptual queries
    long_queries = [
        "how does memory allocation work in the kernel",
        "network packet processing and filtering mechanism",
        "process scheduling algorithm implementation details",
        "file system virtual operations and inode management",
        "interrupt handling and deferred work processing"
    ]
    latencies = []
    for q in long_queries * 20:
        result = search_with_timing(q, endpoint)
        if result["success"]:
            latencies.append(result["latency_ms"])
    results["long_conceptual"] = calculate_percentiles(latencies)

    return results


def benchmark_cold_vs_warm(queries: list, endpoint: str) -> dict:
    """Benchmark cold start vs warm cache performance."""
    # Cold start (first query)
    cold_latencies = []
    for q in queries[:10]:
        result = search_with_timing(q, endpoint)
        if result["success"]:
            cold_latencies.append(result["latency_ms"])

    # Warm cache (repeated queries)
    warm_latencies = []
    for q in queries[:10]:
        for _ in range(5):
            result = search_with_timing(q, endpoint)
            if result["success"]:
                warm_latencies.append(result["latency_ms"])

    return {
        "cold_start": calculate_percentiles(cold_latencies),
        "warm_cache": calculate_percentiles(warm_latencies)
    }


def main():
    parser = argparse.ArgumentParser(description="Benchmark AgentFS search latency")
    parser.add_argument("--data-dir", type=str, required=True, help="Data directory")
    parser.add_argument("--output", type=str, required=True, help="Output JSON file")
    parser.add_argument("--endpoint", type=str, default="http://localhost:8080", help="AgentFS endpoint")
    args = parser.parse_args()

    data_dir = Path(args.data_dir)
    output_file = Path(args.output)

    # Load query set
    queries_file = data_dir / "queries.json"
    if queries_file.exists():
        with open(queries_file) as f:
            query_data = json.load(f)
        queries = [q["query"] for q in query_data.get("keyword", [])]
    else:
        queries = ["mutex", "tcp", "memory allocation", "file system", "network"]

    results = {
        "timestamp": time.strftime("%Y-%m-%d %H:%M:%S"),
        "endpoint": args.endpoint,
        "benchmarks": {}
    }

    # Single query latency
    print("Benchmarking single query latency...")
    results["benchmarks"]["single_query"] = benchmark_single_query_latency(queries, args.endpoint)
    print(f"  P50: {results['benchmarks']['single_query']['percentiles']['p50']:.1f}ms")
    print(f"  P99: {results['benchmarks']['single_query']['percentiles']['p99']:.1f}ms")

    # Concurrent queries
    print("\nBenchmarking concurrent queries...")
    results["benchmarks"]["concurrent"] = {}
    for concurrency in [1, 4, 8, 16]:
        print(f"  Concurrency {concurrency}...")
        result = benchmark_concurrent_queries(queries, args.endpoint, concurrency)
        results["benchmarks"]["concurrent"][f"c{concurrency}"] = result
        print(f"    P50: {result['percentiles']['p50']:.1f}ms, QPS: {result['throughput_qps']:.1f}")

    # Query complexity
    print("\nBenchmarking query complexity...")
    results["benchmarks"]["by_complexity"] = benchmark_query_complexity(args.endpoint)

    # Cold vs warm
    print("\nBenchmarking cold vs warm cache...")
    results["benchmarks"]["cache_effect"] = benchmark_cold_vs_warm(queries, args.endpoint)

    # Save results
    with open(output_file, "w") as f:
        json.dump(results, f, indent=2)

    print(f"\nResults saved to {output_file}")


if __name__ == "__main__":
    main()

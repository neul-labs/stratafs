#!/usr/bin/env python3
"""
Generate LaTeX tables from benchmark results.

Creates publication-ready tables for the research paper.
"""

import argparse
import json
from pathlib import Path


def generate_indexing_table(results: dict) -> str:
    """Generate indexing performance table."""
    lines = [
        "\\begin{table}[h]",
        "\\centering",
        "\\caption{Indexing Performance}",
        "\\label{tab:indexing}",
        "\\begin{tabular}{lrrr}",
        "\\toprule",
        "Corpus & Files & Time (s) & Throughput (files/s) \\\\",
        "\\midrule"
    ]

    benchmarks = results.get("benchmarks", {})

    for name, data in benchmarks.items():
        if name == "scaling":
            continue
        if isinstance(data, dict) and "total_files" in data:
            lines.append(
                f"{name.replace('_', ' ').title()} & "
                f"{data.get('total_files', 'N/A'):,} & "
                f"{data.get('indexing_time_sec', 0):.1f} & "
                f"{data.get('files_per_sec', 0):.1f} \\\\"
            )

    lines.extend([
        "\\bottomrule",
        "\\end{tabular}",
        "\\end{table}"
    ])

    return "\n".join(lines)


def generate_search_quality_table(results: dict) -> str:
    """Generate search quality metrics table."""
    lines = [
        "\\begin{table}[h]",
        "\\centering",
        "\\caption{Search Quality by Query Type}",
        "\\label{tab:search-quality}",
        "\\begin{tabular}{lrrr}",
        "\\toprule",
        "Query Type & P@5 & P@10 & MRR \\\\",
        "\\midrule"
    ]

    by_type = results.get("benchmarks", {}).get("by_query_type", {})

    for query_type, metrics in by_type.items():
        lines.append(
            f"{query_type.title()} & "
            f"{metrics.get('avg_precision_at_5', 0):.3f} & "
            f"{metrics.get('avg_precision_at_10', 0):.3f} & "
            f"{metrics.get('avg_mrr', 0):.3f} \\\\"
        )

    lines.extend([
        "\\bottomrule",
        "\\end{tabular}",
        "\\end{table}"
    ])

    return "\n".join(lines)


def generate_latency_table(results: dict) -> str:
    """Generate latency metrics table."""
    lines = [
        "\\begin{table}[h]",
        "\\centering",
        "\\caption{Search Latency (milliseconds)}",
        "\\label{tab:latency}",
        "\\begin{tabular}{lrrrr}",
        "\\toprule",
        "Configuration & P50 & P95 & P99 & Mean \\\\",
        "\\midrule"
    ]

    # Single query
    single = results.get("benchmarks", {}).get("single_query", {}).get("percentiles", {})
    if single:
        lines.append(
            f"Single Query & "
            f"{single.get('p50', 0):.1f} & "
            f"{single.get('p95', 0):.1f} & "
            f"{single.get('p99', 0):.1f} & "
            f"{single.get('mean', 0):.1f} \\\\"
        )

    # Concurrent
    concurrent = results.get("benchmarks", {}).get("concurrent", {})
    for key, data in concurrent.items():
        p = data.get("percentiles", {})
        lines.append(
            f"Concurrent ({key}) & "
            f"{p.get('p50', 0):.1f} & "
            f"{p.get('p95', 0):.1f} & "
            f"{p.get('p99', 0):.1f} & "
            f"{p.get('mean', 0):.1f} \\\\"
        )

    lines.extend([
        "\\bottomrule",
        "\\end{tabular}",
        "\\end{table}"
    ])

    return "\n".join(lines)


def generate_comparison_table(results: dict) -> str:
    """Generate system comparison table."""
    lines = [
        "\\begin{table}[h]",
        "\\centering",
        "\\caption{System Comparison}",
        "\\label{tab:comparison}",
        "\\begin{tabular}{lcccr}",
        "\\toprule",
        "System & Semantic & Local & Cost & Latency (ms) \\\\",
        "\\midrule"
    ]

    benchmarks = results.get("benchmarks", {})

    systems = [
        ("StrataFS", "stratafs", "\\checkmark", "\\checkmark", "Free"),
        ("ripgrep", "ripgrep", "--", "\\checkmark", "Free"),
        ("Elasticsearch", "elasticsearch", "--", "\\checkmark", "Free"),
        ("OpenAI+FAISS", "openai_faiss", "\\checkmark", "--", "\\$\\$\\$"),
    ]

    for display_name, key, semantic, local, cost in systems:
        data = benchmarks.get(key, {})
        if "error" in data:
            latency = "N/A"
        else:
            latency = f"{data.get('avg_latency_ms', 0):.1f}"
        lines.append(f"{display_name} & {semantic} & {local} & {cost} & {latency} \\\\")

    lines.extend([
        "\\bottomrule",
        "\\end{tabular}",
        "\\end{table}"
    ])

    return "\n".join(lines)


def generate_ablation_table(results: dict) -> str:
    """Generate ablation study table for fusion weights."""
    lines = [
        "\\begin{table}[h]",
        "\\centering",
        "\\caption{Ablation Study: Fusion Weights}",
        "\\label{tab:ablation-fusion}",
        "\\begin{tabular}{lccrr}",
        "\\toprule",
        "Configuration & BM25 & Vector & P@10 & MRR \\\\",
        "\\midrule"
    ]

    fusion = results.get("studies", {}).get("fusion_weights", {})

    for name, data in fusion.items():
        p10 = data.get("precision_at_10", "TBD")
        mrr = data.get("mrr", "TBD")
        lines.append(
            f"{name.replace('_', ' ').title()} & "
            f"{data.get('bm25_weight', 0)} & "
            f"{data.get('vector_weight', 0)} & "
            f"{p10} & {mrr} \\\\"
        )

    lines.extend([
        "\\bottomrule",
        "\\end{tabular}",
        "\\end{table}"
    ])

    return "\n".join(lines)


def main():
    parser = argparse.ArgumentParser(description="Generate LaTeX tables")
    parser.add_argument("--results-dir", type=str, required=True, help="Results directory")
    parser.add_argument("--output", type=str, required=True, help="Output LaTeX file")
    args = parser.parse_args()

    results_dir = Path(args.results_dir)
    output_file = Path(args.output)

    tables = []

    # Header
    tables.append("% Auto-generated LaTeX tables for StrataFS paper")
    tables.append(f"% Generated from results in {results_dir}")
    tables.append("")

    # Indexing table
    indexing_file = results_dir / "indexing_results.json"
    if indexing_file.exists():
        with open(indexing_file) as f:
            data = json.load(f)
        tables.append(generate_indexing_table(data))
        tables.append("")

    # Search quality table
    quality_file = results_dir / "search_quality_results.json"
    if quality_file.exists():
        with open(quality_file) as f:
            data = json.load(f)
        tables.append(generate_search_quality_table(data))
        tables.append("")

    # Latency table
    latency_file = results_dir / "latency_results.json"
    if latency_file.exists():
        with open(latency_file) as f:
            data = json.load(f)
        tables.append(generate_latency_table(data))
        tables.append("")

    # Comparison table
    comparison_file = results_dir / "comparison_results.json"
    if comparison_file.exists():
        with open(comparison_file) as f:
            data = json.load(f)
        tables.append(generate_comparison_table(data))
        tables.append("")

    # Ablation table
    ablation_file = results_dir / "ablation_results.json"
    if ablation_file.exists():
        with open(ablation_file) as f:
            data = json.load(f)
        tables.append(generate_ablation_table(data))
        tables.append("")

    # Write output
    with open(output_file, "w") as f:
        f.write("\n".join(tables))

    print(f"LaTeX tables written to {output_file}")
    print(f"Tables generated: {len([t for t in tables if t.startswith('\\\\begin')])}")


if __name__ == "__main__":
    main()

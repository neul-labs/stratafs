#!/bin/bash
# StrataFS Research Benchmarks
# Run all benchmarks and generate results for the paper

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESULTS_DIR="$SCRIPT_DIR/results"
DATA_DIR="$SCRIPT_DIR/data"

# Create directories
mkdir -p "$RESULTS_DIR"
mkdir -p "$DATA_DIR"

echo "=== StrataFS Research Benchmarks ==="
echo "Results will be saved to: $RESULTS_DIR"
echo ""

# Check for OpenAI key (optional, for comparison benchmarks)
if [ -n "$OPENAI_API_KEY" ]; then
    echo "OpenAI API key found - will run OpenAI embedding comparisons"
else
    echo "No OPENAI_API_KEY - skipping OpenAI embedding comparisons"
fi

# 1. Download/prepare datasets
echo ""
echo "=== Preparing Datasets ==="
python3 "$SCRIPT_DIR/prepare_datasets.py" --output "$DATA_DIR"

# 2. Run indexing benchmarks
echo ""
echo "=== Running Indexing Benchmarks ==="
python3 "$SCRIPT_DIR/benchmark_indexing.py" \
    --data-dir "$DATA_DIR" \
    --output "$RESULTS_DIR/indexing_results.json"

# 3. Run search quality benchmarks
echo ""
echo "=== Running Search Quality Benchmarks ==="
python3 "$SCRIPT_DIR/benchmark_search_quality.py" \
    --data-dir "$DATA_DIR" \
    --output "$RESULTS_DIR/search_quality_results.json"

# 4. Run latency benchmarks
echo ""
echo "=== Running Latency Benchmarks ==="
python3 "$SCRIPT_DIR/benchmark_latency.py" \
    --data-dir "$DATA_DIR" \
    --output "$RESULTS_DIR/latency_results.json"

# 5. Run ablation studies
echo ""
echo "=== Running Ablation Studies ==="
python3 "$SCRIPT_DIR/benchmark_ablation.py" \
    --data-dir "$DATA_DIR" \
    --output "$RESULTS_DIR/ablation_results.json"

# 6. Run comparison benchmarks (ripgrep, elasticsearch if available)
echo ""
echo "=== Running Comparison Benchmarks ==="
python3 "$SCRIPT_DIR/benchmark_comparisons.py" \
    --data-dir "$DATA_DIR" \
    --output "$RESULTS_DIR/comparison_results.json"

# 7. Generate LaTeX tables
echo ""
echo "=== Generating LaTeX Tables ==="
python3 "$SCRIPT_DIR/generate_latex.py" \
    --results-dir "$RESULTS_DIR" \
    --output "$RESULTS_DIR/tables.tex"

echo ""
echo "=== Benchmarks Complete ==="
echo "Results saved to: $RESULTS_DIR"
echo ""
echo "To update the paper, copy values from:"
echo "  $RESULTS_DIR/tables.tex"

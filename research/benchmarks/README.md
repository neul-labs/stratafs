# StrataFS Benchmark Suite

Benchmarks for the StrataFS research paper.

## Quick Start

```bash
# Install dependencies
make install

# With OpenAI comparison support
make install-openai

# Run everything
make all
```

## Individual Commands

```bash
make prepare          # Download datasets
make benchmarks       # Run all benchmarks
make latex            # Generate LaTeX tables

# Individual benchmarks
make indexing
make search-quality
make latency
make ablation
make comparisons
```

## Requirements

- Python 3.9+
- Poetry
- StrataFS running at localhost:8080 (for search benchmarks)
- Optional: `OPENAI_API_KEY` for embedding comparisons

## Output

Results are saved to `./results/` as JSON files. Run `make latex` to generate publication-ready tables for the paper.

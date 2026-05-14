# Makefile for StrataFS benchmarks
# Uses Poetry for Python dependency management

SHELL := /bin/bash
POETRY := poetry
PYTHON := $(POETRY) run python3

DATA_DIR := ./data
RESULTS_DIR := ./results

.PHONY: all install clean prepare benchmarks indexing search-quality latency ablation comparisons latex help

all: install prepare benchmarks latex

help:
	@echo "StrataFS Benchmark Suite"
	@echo ""
	@echo "Usage:"
	@echo "  make install          Install dependencies with Poetry"
	@echo "  make install-openai   Install with OpenAI support"
	@echo "  make prepare          Download/generate benchmark datasets"
	@echo "  make benchmarks       Run all benchmarks"
	@echo "  make latex            Generate LaTeX tables from results"
	@echo "  make clean            Remove generated data and results"
	@echo ""
	@echo "Individual benchmarks:"
	@echo "  make indexing         Run indexing benchmarks"
	@echo "  make search-quality   Run search quality benchmarks"
	@echo "  make latency          Run latency benchmarks"
	@echo "  make ablation         Run ablation studies"
	@echo "  make comparisons      Run comparison benchmarks"

install:
	$(POETRY) install --no-root

install-openai:
	$(POETRY) install --no-root --extras openai

$(DATA_DIR):
	mkdir -p $(DATA_DIR)

$(RESULTS_DIR):
	mkdir -p $(RESULTS_DIR)

prepare: install $(DATA_DIR)
	@echo "=== Preparing Datasets ==="
	$(PYTHON) prepare_datasets.py --output $(DATA_DIR)

benchmarks: indexing search-quality latency ablation comparisons

indexing: install $(RESULTS_DIR)
	@echo "=== Running Indexing Benchmarks ==="
	$(PYTHON) benchmark_indexing.py \
		--data-dir $(DATA_DIR) \
		--output $(RESULTS_DIR)/indexing_results.json

search-quality: install $(RESULTS_DIR)
	@echo "=== Running Search Quality Benchmarks ==="
	$(PYTHON) benchmark_search_quality.py \
		--data-dir $(DATA_DIR) \
		--output $(RESULTS_DIR)/search_quality_results.json

latency: install $(RESULTS_DIR)
	@echo "=== Running Latency Benchmarks ==="
	$(PYTHON) benchmark_latency.py \
		--data-dir $(DATA_DIR) \
		--output $(RESULTS_DIR)/latency_results.json

ablation: install $(RESULTS_DIR)
	@echo "=== Running Ablation Studies ==="
	$(PYTHON) benchmark_ablation.py \
		--data-dir $(DATA_DIR) \
		--output $(RESULTS_DIR)/ablation_results.json

comparisons: install $(RESULTS_DIR)
	@echo "=== Running Comparison Benchmarks ==="
	$(PYTHON) benchmark_comparisons.py \
		--data-dir $(DATA_DIR) \
		--output $(RESULTS_DIR)/comparison_results.json

latex: install $(RESULTS_DIR)
	@echo "=== Generating LaTeX Tables ==="
	$(PYTHON) generate_latex.py \
		--results-dir $(RESULTS_DIR) \
		--output $(RESULTS_DIR)/tables.tex
	@echo ""
	@echo "LaTeX tables saved to $(RESULTS_DIR)/tables.tex"
	@echo "Copy these into ../stratafs-paper.tex"

clean:
	rm -rf $(DATA_DIR) $(RESULTS_DIR)
	rm -rf .venv __pycache__ *.pyc

clean-results:
	rm -rf $(RESULTS_DIR)

# Skip kernel download for faster iteration
prepare-quick: install $(DATA_DIR)
	@echo "=== Preparing Datasets (skipping kernel) ==="
	$(PYTHON) prepare_datasets.py --output $(DATA_DIR) --skip-kernel

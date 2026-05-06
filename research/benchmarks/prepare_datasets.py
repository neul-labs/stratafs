#!/usr/bin/env python3
"""
Prepare datasets for AgentFS benchmarks.

Downloads/generates:
1. Linux Kernel source (subset)
2. Documentation corpus from GitHub
3. Synthetic enterprise data
"""

import argparse
import json
import os
import random
import subprocess
import urllib.request
from pathlib import Path


def download_linux_kernel(output_dir: Path, max_files: int = 10000):
    """Download a subset of Linux kernel source."""
    kernel_dir = output_dir / "linux-kernel"

    if kernel_dir.exists() and len(list(kernel_dir.rglob("*.c"))) > 1000:
        print(f"Linux kernel already exists at {kernel_dir}")
        return kernel_dir

    print("Downloading Linux kernel source (this may take a while)...")
    kernel_dir.mkdir(parents=True, exist_ok=True)

    # Clone with depth=1 for speed
    subprocess.run([
        "git", "clone", "--depth=1",
        "https://github.com/torvalds/linux.git",
        str(kernel_dir)
    ], check=True)

    # Count files
    c_files = list(kernel_dir.rglob("*.c"))
    h_files = list(kernel_dir.rglob("*.h"))
    print(f"Downloaded {len(c_files)} .c files and {len(h_files)} .h files")

    return kernel_dir


def download_docs_corpus(output_dir: Path):
    """Download documentation from popular open-source projects."""
    docs_dir = output_dir / "docs-corpus"

    if docs_dir.exists() and len(list(docs_dir.rglob("*.md"))) > 100:
        print(f"Docs corpus already exists at {docs_dir}")
        return docs_dir

    print("Downloading documentation corpus...")
    docs_dir.mkdir(parents=True, exist_ok=True)

    # Clone docs from popular projects
    repos = [
        ("https://github.com/golang/go.git", "go-docs"),
        ("https://github.com/rust-lang/rust.git", "rust-docs"),
        ("https://github.com/python/cpython.git", "python-docs"),
    ]

    for url, name in repos:
        repo_dir = docs_dir / name
        if not repo_dir.exists():
            print(f"  Cloning {name}...")
            subprocess.run([
                "git", "clone", "--depth=1", "--filter=blob:none",
                "--sparse", url, str(repo_dir)
            ], check=True, capture_output=True)

            # Sparse checkout only docs
            subprocess.run([
                "git", "-C", str(repo_dir), "sparse-checkout", "set", "doc", "docs", "Documentation"
            ], check=True, capture_output=True)

    md_files = list(docs_dir.rglob("*.md"))
    print(f"Downloaded {len(md_files)} markdown files")

    return docs_dir


def generate_enterprise_data(output_dir: Path, num_issues: int = 500):
    """Generate synthetic enterprise data (code + docs + Jira issues)."""
    enterprise_dir = output_dir / "enterprise-sim"

    if enterprise_dir.exists():
        print(f"Enterprise data already exists at {enterprise_dir}")
        return enterprise_dir

    print("Generating synthetic enterprise data...")
    enterprise_dir.mkdir(parents=True, exist_ok=True)

    # Generate synthetic Jira issues
    issues_dir = enterprise_dir / "jira-issues"
    issues_dir.mkdir(exist_ok=True)

    issue_types = ["Bug", "Feature", "Task", "Story"]
    priorities = ["Critical", "High", "Medium", "Low"]
    statuses = ["Open", "In Progress", "Review", "Done"]

    components = [
        "Authentication", "Database", "API", "Frontend", "Backend",
        "Infrastructure", "Security", "Performance", "Testing"
    ]

    descriptions = [
        "User authentication fails when using SSO with special characters in username",
        "Implement rate limiting for API endpoints to prevent abuse",
        "Database queries are slow when filtering by date range",
        "Add dark mode support to the frontend application",
        "Refactor payment processing module for better error handling",
        "Memory leak detected in worker process after extended operation",
        "Add support for bulk import of user data via CSV",
        "Implement caching layer for frequently accessed resources",
        "Security audit findings need to be addressed",
        "Performance degradation observed under high load",
    ]

    for i in range(num_issues):
        issue_key = f"PROJ-{i+1}"
        issue = {
            "key": issue_key,
            "summary": random.choice(descriptions)[:60] + f" (#{i+1})",
            "description": random.choice(descriptions) + "\n\n" +
                          "Additional context and reproduction steps would go here. " * 3,
            "type": random.choice(issue_types),
            "priority": random.choice(priorities),
            "status": random.choice(statuses),
            "component": random.choice(components),
            "labels": random.sample(["urgent", "technical-debt", "customer-reported", "regression", "documentation"],
                                   k=random.randint(0, 3)),
            "created": f"2024-{random.randint(1,12):02d}-{random.randint(1,28):02d}",
            "updated": f"2024-{random.randint(1,12):02d}-{random.randint(1,28):02d}",
        }

        # Write as markdown (like Jira connector does)
        md_content = f"""# {issue_key}: {issue['summary']}

## Metadata

- **Type**: {issue['type']}
- **Priority**: {issue['priority']}
- **Status**: {issue['status']}
- **Component**: {issue['component']}
- **Labels**: {', '.join(issue['labels']) if issue['labels'] else 'None'}
- **Created**: {issue['created']}
- **Updated**: {issue['updated']}

## Description

{issue['description']}

## Comments

### John Doe - 2024-01-15

Looking into this issue. Initial investigation suggests the problem is in the {issue['component'].lower()} module.

### Jane Smith - 2024-01-16

I can confirm this issue. Here's a workaround for now...
"""

        with open(issues_dir / f"{issue_key}.md", "w") as f:
            f.write(md_content)

    # Generate some synthetic code files
    code_dir = enterprise_dir / "src"
    code_dir.mkdir(exist_ok=True)

    for component in components:
        comp_dir = code_dir / component.lower()
        comp_dir.mkdir(exist_ok=True)

        # Generate a few files per component
        for j in range(random.randint(3, 8)):
            filename = f"{component.lower()}_{j}.go"
            content = f"""package {component.lower()}

// {component} module - file {j}
// This is synthetic code for benchmarking purposes

import (
    "context"
    "errors"
    "log"
)

// {component}Service handles {component.lower()} operations
type {component}Service struct {{
    db     Database
    cache  Cache
    logger *log.Logger
}}

// New{component}Service creates a new service instance
func New{component}Service(db Database, cache Cache) *{component}Service {{
    return &{component}Service{{
        db:    db,
        cache: cache,
    }}
}}

// Process handles the main {component.lower()} logic
func (s *{component}Service) Process(ctx context.Context, input string) (string, error) {{
    if input == "" {{
        return "", errors.New("input cannot be empty")
    }}

    // Check cache first
    if cached, ok := s.cache.Get(input); ok {{
        return cached, nil
    }}

    // Process and store result
    result := s.doProcess(input)
    s.cache.Set(input, result)

    return result, nil
}}

func (s *{component}Service) doProcess(input string) string {{
    // Implementation details for {component.lower()}
    return "processed: " + input
}}
"""
            with open(comp_dir / filename, "w") as f:
                f.write(content)

    print(f"Generated {num_issues} Jira issues and code files")
    return enterprise_dir


def generate_query_set(output_dir: Path):
    """Generate evaluation query set with ground truth."""
    queries_file = output_dir / "queries.json"

    queries = {
        "keyword": [
            {"query": "mutex_lock", "type": "keyword", "expected_files": ["kernel/locking/"]},
            {"query": "tcp_sendmsg", "type": "keyword", "expected_files": ["net/ipv4/"]},
            {"query": "kmalloc", "type": "keyword", "expected_files": ["mm/", "include/linux/slab.h"]},
            {"query": "printk", "type": "keyword", "expected_files": ["kernel/printk/"]},
            {"query": "schedule", "type": "keyword", "expected_files": ["kernel/sched/"]},
        ] * 8,  # 40 keyword queries

        "conceptual": [
            {"query": "how does memory allocation work", "type": "conceptual",
             "relevant_concepts": ["kmalloc", "vmalloc", "slab", "page allocation"]},
            {"query": "network packet processing", "type": "conceptual",
             "relevant_concepts": ["sk_buff", "netfilter", "tcp", "socket"]},
            {"query": "process scheduling algorithm", "type": "conceptual",
             "relevant_concepts": ["cfs", "scheduler", "runqueue", "priority"]},
            {"query": "file system operations", "type": "conceptual",
             "relevant_concepts": ["vfs", "inode", "dentry", "superblock"]},
            {"query": "interrupt handling mechanism", "type": "conceptual",
             "relevant_concepts": ["irq", "handler", "softirq", "tasklet"]},
        ] * 8,  # 40 conceptual queries

        "hybrid": [
            {"query": "mutex implementation for SMP", "type": "hybrid",
             "keywords": ["mutex", "smp"], "concepts": ["locking", "synchronization"]},
            {"query": "TCP congestion control algorithms", "type": "hybrid",
             "keywords": ["tcp", "congestion"], "concepts": ["networking", "flow control"]},
        ] * 10,  # 20 hybrid queries
    }

    with open(queries_file, "w") as f:
        json.dump(queries, f, indent=2)

    print(f"Generated query set at {queries_file}")
    return queries_file


def main():
    parser = argparse.ArgumentParser(description="Prepare benchmark datasets")
    parser.add_argument("--output", type=str, required=True, help="Output directory")
    parser.add_argument("--skip-kernel", action="store_true", help="Skip Linux kernel download")
    args = parser.parse_args()

    output_dir = Path(args.output)
    output_dir.mkdir(parents=True, exist_ok=True)

    # Prepare datasets
    if not args.skip_kernel:
        download_linux_kernel(output_dir)

    download_docs_corpus(output_dir)
    generate_enterprise_data(output_dir)
    generate_query_set(output_dir)

    # Write dataset manifest
    manifest = {
        "linux_kernel": str(output_dir / "linux-kernel"),
        "docs_corpus": str(output_dir / "docs-corpus"),
        "enterprise_sim": str(output_dir / "enterprise-sim"),
        "queries": str(output_dir / "queries.json"),
    }

    with open(output_dir / "manifest.json", "w") as f:
        json.dump(manifest, f, indent=2)

    print(f"\nDatasets prepared in {output_dir}")
    print(f"Manifest: {output_dir / 'manifest.json'}")


if __name__ == "__main__":
    main()

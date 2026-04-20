# Configuration Guide

AgentFS uses a JSON-based configuration system stored in `~/.agentfs/config.json` that provides comprehensive control over all aspects of the system.

## Quick Start

```bash
# Initialize default configuration
./build/agentfs config init

# View current configuration
./build/agentfs config show

# List configured storage sources
./build/agentfs source list

# Start AgentFS
./build/agentfs
```

## Configuration Structure

```json
{
  "version": "0.2.0",
  "agent_dir": ".agentfs",
  "global_dir": "/home/user/.agentfs",
  "sources": [
    {
      "id": "default-local",
      "name": "Current Directory",
      "type": "local",
      "enabled": true,
      "path": "/path/to/directory",
      "local_cache_dir": "",
      "credentials": null,
      "filters": {
        "include_patterns": ["*"],
        "exclude_patterns": [".git/**", "node_modules/**"],
        "max_file_size": 104857600,
        "ignore_hidden": true
      }
    }
  ],
  "server": {
    "api_port": 8080,
    "mcp_port": 8081
  },
  "worker": {
    "count": 4,
    "scan_interval": "10s",
    "batch_size": 10
  },
  "embedding": {
    "model": "bge-base-en-v1.5",
    "cache_dir": "/home/user/.agentfs/fastembed_cache",
    "dimension": 768,
    "model_info": {
      "name": "BGE Base EN v1.5",
      "description": "Improved BAAI General Embedding with better performance",
      "max_tokens": 512,
      "file_size": "420MB",
      "speed": "medium",
      "quality": "high",
      "language": "English"
    },
    "performance": {
      "batch_size": 32,
      "max_concurrency": 4,
      "cache_results": true,
      "enable_gpu": false
    }
  },
  "database": {
    "compression_enabled": true,
    "compression_threshold": 512,
    "maintenance_interval": "24h",
    "deleted_threshold": "168h"
  }
}
```

## Server Configuration

Configure API and MCP server ports:

```json
{
  "server": {
    "api_port": 8080,    // REST API server port
    "mcp_port": 8081     // Model Context Protocol server port
  }
}
```

## Worker Configuration

Control file processing and scanning behavior:

```json
{
  "worker": {
    "count": 4,               // Number of concurrent workers
    "scan_interval": "10s",   // Remote source scan interval
    "batch_size": 10          // Files processed per batch
  }
}
```

## Embedding Models

Configure AI embedding models:

```json
{
  "embedding": {
    "model": "bge-base-en-v1.5",  // Model identifier
    "dimension": 768,              // Vector dimensions
    "cache_dir": "~/.agentfs/fastembed_cache"
  }
}
```

Available models:
- `bge-base-en-v1.5` (768d) - Recommended, best balance
- `bge-small-en-v1.5` (384d) - Faster, smaller
- `bge-base-en` (768d) - Original BGE base
- `bge-small-en` (384d) - Original BGE small

## Database Settings

Control storage optimization:

```json
{
  "database": {
    "compression_enabled": true,      // Enable text compression
    "compression_threshold": 512,     // Compress chunks > 512 bytes
    "maintenance_interval": "24h",    // Auto-maintenance frequency
    "deleted_threshold": "168h"       // Cleanup deleted files after 7 days
  }
}
```

## File Filters

Configure which files to index:

```json
{
  "filters": {
    "include_patterns": ["*"],                    // Include all by default
    "exclude_patterns": [                         // Exclude these patterns
      ".git/**",
      "node_modules/**",
      "*.tmp",
      "*.log"
    ],
    "max_file_size": 104857600,                  // Max 100MB files
    "ignore_hidden": true                        // Skip hidden files
  }
}
```

## Environment Variable Overrides

Runtime configuration overrides:

- `AGENTFS_GLOBAL_DIR` - Override global directory location
- `AGENTFS_API_PORT` - Override API server port
- `AGENTFS_MCP_PORT` - Override MCP server port
- `AGENTFS_WORKER_COUNT` - Override worker count
- `AGENTFS_EMBEDDING_MODEL` - Override embedding model

Example:
```bash
export AGENTFS_API_PORT=9000
export AGENTFS_WORKER_COUNT=8
./build/agentfs
```

## CLI Commands

### Configuration Management
```bash
# Initialize with defaults
agentfs config init

# Show current configuration
agentfs config show
```

### Source Management
```bash
# List all sources
agentfs source list

# Add new source (interactive)
agentfs source add
```

## Configuration Validation

AgentFS validates configuration on startup:

✅ **Storage Credentials** - Verify access to configured sources
✅ **Directory Access** - Check read/write permissions
✅ **Model Availability** - Validate embedding model access
✅ **Port Availability** - Ensure ports are not in use
✅ **Filter Patterns** - Validate glob pattern syntax

Validation errors include helpful suggestions for resolution.
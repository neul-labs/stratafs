# Architecture Overview

AgentFS implements a modular, multi-storage architecture designed for scalability, extensibility, and AI agent integration.

## High-Level Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   REST API      │    │   MCP Server    │    │   CLI Tools     │
│   (Port 8080)   │    │   (Port 8081)   │    │                 │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                    ┌─────────────────┐
                    │  Search Engine  │
                    │  (Hybrid FTS5   │
                    │  + Vector)      │
                    └─────────┬───────┘
                              │
              ┌───────────────┼───────────────┐
              │               │               │
    ┌─────────▼───────┐ ┌─────▼─────┐ ┌───────▼───────┐
    │   Database      │ │ Embedder  │ │  Job Queue    │
    │   (SQLite +     │ │ (FastEmbed│ │  (Processing) │
    │   Vector Index) │ │  +ONNX)   │ │               │
    └─────────────────┘ └───────────┘ └───────┬───────┘
                                              │
                                    ┌─────────▼───────┐
                                    │    Monitor      │
                                    │ (File Watcher + │
                                    │ Remote Scanner) │
                                    └─────────┬───────┘
                                              │
                                ┌─────────────▼─────────────┐
                                │     Storage Factory       │
                                │  (Multi-Backend Support)  │
                                └─────────────┬─────────────┘
                                              │
                ┌─────────────────────────────┼─────────────────────────────┐
                │                             │                             │
      ┌─────────▼─────────┐         ┌─────────▼─────────┐         ┌─────────▼─────────┐
      │  Local FileSystem │         │   Cloud Storage   │         │   Future Backends │
      │   (Real-time)     │         │ (S3, GCS, Azure)  │         │                   │
      └───────────────────┘         └───────────────────┘         └───────────────────┘
```

## Core Components

### 1. Storage Layer

#### Storage Factory
- **Purpose**: Unified interface for all storage backends
- **Pattern**: Factory pattern with strategy for different storage types
- **Responsibilities**:
  - Backend selection and instantiation
  - Credential validation
  - Connection pooling and management

#### Filesystem Abstraction
- **Interface**: Common operations (Open, Walk, Stat) across all backends
- **Backends**: Local, S3, GCS, Azure, and extensible for future storage
- **Features**:
  - Transparent caching for remote sources
  - Error handling and retry logic
  - Bandwidth optimization

### 2. Processing Pipeline

#### File Monitor
- **Local Sources**: Real-time filesystem events via fsnotify
- **Remote Sources**: Periodic scanning with change detection
- **Features**:
  - Concurrent processing
  - File filtering and pattern matching
  - Duplicate detection and handling

#### Job Queue
- **Purpose**: Asynchronous file processing pipeline
- **Storage**: SQLite-based queue with persistence
- **Job Types**: Parse, Embed, Index, Cleanup
- **Features**:
  - Priority-based scheduling
  - Retry logic and failure handling
  - Worker pool management

#### File Parsers
- **Architecture**: Registry-based modular system
- **Supported Types**: Text, Markdown, PDF, DOCX, XLSX, code files
- **Extension**: Plugin-like architecture for new formats
- **Features**:
  - Content extraction and normalization
  - Metadata preservation
  - Error recovery

### 3. AI/ML Layer

#### Embedding System
- **Engine**: FastEmbed-go with ONNX Runtime
- **Models**: BGE family (base/small, various dimensions)
- **Features**:
  - Model caching and optimization
  - Batch processing for efficiency
  - Cross-platform compatibility

#### Text Processing
- **Chunking**: Intelligent text segmentation
- **Strategies**: Sliding window, sentence boundaries, semantic chunks
- **Optimization**: Overlap handling, size limits

### 4. Search Engine

#### Hybrid Search
- **FTS5**: Full-text search with SQLite FTS5 extension
- **Vector Search**: Similarity search using embeddings
- **Fusion**: Relevance score combination and ranking

#### Index Management
- **Storage**: SQLite databases with custom indexes
- **Optimization**: Automatic index maintenance and compaction
- **Scaling**: Per-source database isolation

### 5. API Layer

#### REST API
- **Framework**: Standard library HTTP with custom routing
- **Features**: Search, document retrieval, statistics
- **Response Format**: JSON with structured error handling

#### MCP Server
- **Protocol**: Model Context Protocol for AI agents
- **Optimization**: Agent-specific response formatting
- **Tools**: Structured tool execution interface

## Data Flow

### Local File Processing
```
File Change Event → File Watcher → Job Queue → Parser → Embedder → Database → Search Index
                                      ↓
                                 File Cleanup (if remote)
```

### Remote File Processing
```
Scan Timer → Remote Scanner → Download to Cache → Job Queue → Parser → Embedder → Database → Search Index → Cache Cleanup
```

### Search Query Flow
```
API Request → Search Engine → FTS5 Query + Vector Query → Result Fusion → Response Formatting → API Response
```

## Design Principles

### 1. Read-Only Source Integrity
- **No Modifications**: Source files and directories remain untouched
- **Cache Management**: Temporary local copies for remote processing
- **Metadata Isolation**: All AgentFS data in separate `.agentfs` directories

### 2. Modular Architecture
- **Interfaces**: Clear contracts between components
- **Dependency Injection**: Configurable component dependencies
- **Extension Points**: Plugin architectures for parsers and storage

### 3. Scalability
- **Horizontal**: Multiple workers, concurrent processing
- **Vertical**: Efficient algorithms, memory management
- **Storage**: Per-source database isolation, compression

### 4. Fault Tolerance
- **Graceful Degradation**: Continue processing despite individual file failures
- **Retry Logic**: Automatic retry for transient failures
- **State Recovery**: Persistent queue state across restarts

## Configuration Management

### JSON-Based Configuration
```json
{
  "sources": [],     // Storage source definitions
  "server": {},      // API server configuration
  "worker": {},      // Processing configuration
  "embedding": {},   // AI model configuration
  "database": {}     // Storage optimization
}
```

### Environment Overrides
- Runtime configuration via environment variables
- Deployment-specific settings without config file changes
- Sensitive credential handling

### Validation Pipeline
- Startup validation of all configuration
- Credential verification
- Resource availability checks

## Security Considerations

### Data Privacy
- **Local Processing**: Embeddings generated locally
- **No External Calls**: No data sent to external AI services
- **Isolation**: Each source gets isolated database

### Access Control
- **File System Permissions**: Respects OS-level access controls
- **API Security**: Local-only by default, proxy for remote access
- **Credential Management**: Secure storage of cloud credentials

### Network Security
- **Outbound Only**: Only connects to configured cloud storage
- **TLS**: Encrypted connections to cloud services
- **Firewall Friendly**: Configurable ports, local binding

## Performance Characteristics

### Memory Usage
- **Streaming Processing**: Files processed in chunks, not loaded entirely
- **Embedding Caching**: LRU cache for frequently accessed embeddings
- **Database Optimization**: Compression, indexing, maintenance

### CPU Utilization
- **Worker Pools**: Configurable concurrency levels
- **Batch Processing**: Efficient embedding generation
- **Background Tasks**: Maintenance and cleanup operations

### Storage Optimization
- **Compression**: Automatic text compression (40-60% savings)
- **Deduplication**: Avoid reprocessing unchanged files
- **Maintenance**: Automatic cleanup and optimization

### Network Efficiency
- **Change Detection**: Only download modified files
- **Caching**: Local cache for remote file processing
- **Batching**: Efficient API usage for cloud storage

## Extension Points

### New Storage Backends
1. Implement `filesystem.FileSystem` interface
2. Add factory method in `storage.Factory`
3. Add configuration schema
4. Implement credential validation

### New File Parsers
1. Implement `parsers.Parser` interface
2. Register with `parsers.Registry`
3. Add extension mapping
4. Handle parsing errors gracefully

### New Embedding Models
1. Add model configuration
2. Implement model loading
3. Handle dimension changes
4. Test compatibility

### API Extensions
1. Add new endpoints to REST API
2. Implement MCP tools
3. Maintain backward compatibility
4. Add appropriate error handling

## Monitoring and Observability

### Metrics
- Processing rates and queue depths
- Search performance and accuracy
- Storage usage and optimization
- API response times and error rates

### Logging
- Structured logging with levels
- Component-specific log namespaces
- Performance and error tracking
- Debug mode for troubleshooting

### Health Checks
- Component health status
- Dependency availability
- Resource utilization
- Configuration validation

This architecture enables AgentFS to scale from single-user local installations to multi-tenant cloud deployments while maintaining simplicity and performance.
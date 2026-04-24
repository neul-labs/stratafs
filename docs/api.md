# API Reference

AgentFS provides two main APIs for accessing your indexed content: a REST API for general purpose access and a Model Context Protocol (MCP) server for AI agent integration.

## REST API (Port 8080)

The REST API provides comprehensive search and management capabilities.

### Base URL
```
http://localhost:8080
```

### Endpoints

#### Health Check
```http
GET /health
```

Returns system health and status information.

**Response:**
```json
{
  "status": "healthy",
  "version": "0.2.0",
  "uptime": "2h15m30s",
  "sources": 3,
  "indexed_files": 1250,
  "storage_used": "245MB"
}
```

#### Hybrid Search
```http
GET /search?q={query}&limit={limit}&offset={offset}
```

Performs hybrid search combining full-text search with vector similarity.

**Parameters:**
- `q` (required) - Search query (natural language or keywords)
- `limit` (optional) - Maximum results to return (default: 10, max: 100)
- `offset` (optional) - Pagination offset (default: 0)
- `sources` (optional) - Comma-separated source IDs to search

**Example:**
```bash
curl "http://localhost:8080/search?q=machine learning algorithms&limit=5"
```

**Response:**
```json
{
  "query": "machine learning algorithms",
  "total_results": 42,
  "results": [
    {
      "file_path": "/docs/ml/neural-networks.md",
      "source_name": "Documentation",
      "chunk_content": "Neural networks are a class of machine learning algorithms...",
      "relevance_score": 0.95,
      "chunk_offset": 1024,
      "chunk_length": 256,
      "chunk_strategy": "sentence",
      "file_size": 15420,
      "modified_time": "2024-01-15T10:30:00Z"
    }
  ],
  "search_time_ms": 45
}
```

#### Document Retrieval
```http
GET /documents/{path}
```

Retrieve full document content by file path.

**Example:**
```bash
curl "http://localhost:8080/documents/docs%2Fml%2Fneural-networks.md"
```

**Response:**
```json
{
  "file_path": "/docs/ml/neural-networks.md",
  "source_name": "Documentation",
  "content": "# Neural Networks\n\nNeural networks are...",
  "file_size": 15420,
  "modified_time": "2024-01-15T10:30:00Z",
  "chunks": 12,
  "chunking_strategy": "sentence",
  "processing_stats": {
    "chunks_generated": 12,
    "avg_chunk_size": 284,
    "overlap_size": 50
  }
}
```

#### Queue Statistics
```http
GET /queue/stats
```

View processing queue statistics and system performance.

**Response:**
```json
{
  "pending_jobs": 5,
  "processing_jobs": 2,
  "completed_jobs": 1248,
  "failed_jobs": 3,
  "worker_count": 4,
  "average_processing_time_ms": 250
}
```

#### Chunking Statistics
```http
GET /chunking/stats
```

View chunking strategy usage and performance statistics.

**Response:**
```json
{
  "strategies": {
    "simple": {
      "files_processed": 450,
      "avg_chunks_per_file": 8.5,
      "avg_processing_time_ms": 120
    },
    "sentence": {
      "files_processed": 200,
      "avg_chunks_per_file": 12.3,
      "avg_processing_time_ms": 180
    },
    "separator": {
      "files_processed": 600,
      "avg_chunks_per_file": 15.2,
      "avg_processing_time_ms": 95
    }
  },
  "total_chunks": 12450,
  "compression_stats": {
    "compressed_chunks": 8100,
    "compression_ratio": 0.65,
    "storage_saved": "45MB"
  }
}
```

#### Source Statistics
```http
GET /sources/stats
```

View statistics for all configured storage sources.

**Response:**
```json
{
  "sources": [
    {
      "id": "local-docs",
      "name": "Documentation",
      "type": "local",
      "status": "active",
      "indexed_files": 450,
      "total_size": "85MB",
      "last_scan": "2024-01-15T10:30:00Z"
    },
    {
      "id": "s3-archive",
      "name": "S3 Archive",
      "type": "s3",
      "status": "scanning",
      "indexed_files": 800,
      "total_size": "160MB",
      "last_scan": "2024-01-15T10:25:00Z"
    }
  ]
}
```

## Model Context Protocol (Port 8081)

The MCP server provides AI-optimized endpoints for seamless agent integration.

### Base URL
```
http://localhost:8081
```

### Endpoints

#### Protocol Information
```http
GET /mcp
```

Returns MCP protocol capabilities and server information.

**Response:**
```json
{
  "protocol_version": "1.0",
  "server_name": "agentfs-mcp",
  "capabilities": {
    "search": true,
    "resources": true,
    "tools": ["search", "retrieve", "stats"]
  },
  "description": "AgentFS Model Context Protocol Server"
}
```

#### AI-Optimized Search
```http
GET /mcp/search?q={query}&context={context}
```

Performs search optimized for AI agent consumption with enhanced context.

**Parameters:**
- `q` (required) - Search query
- `context` (optional) - Additional context for relevance scoring
- `max_results` (optional) - Maximum results (default: 5, max: 20)

**Example:**
```bash
curl "http://localhost:8081/mcp/search?q=API authentication&context=web development"
```

**Response:**
```json
{
  "query": "API authentication",
  "context": "web development",
  "results": [
    {
      "resource_id": "docs/api/auth.md#jwt-tokens",
      "title": "JWT Token Authentication",
      "content": "JWT tokens provide stateless authentication...",
      "relevance": 0.92,
      "metadata": {
        "file_type": "markdown",
        "section": "JWT Tokens",
        "source": "API Documentation"
      }
    }
  ],
  "suggested_actions": [
    "Show JWT implementation example",
    "Explain token validation process"
  ]
}
```

#### Resource Discovery
```http
GET /mcp/resources
```

List available resources and their capabilities.

**Response:**
```json
{
  "resources": [
    {
      "id": "search",
      "name": "Semantic Search",
      "description": "Search across all indexed content",
      "type": "tool"
    },
    {
      "id": "retrieve",
      "name": "Document Retrieval",
      "description": "Retrieve full document content",
      "type": "tool"
    }
  ]
}
```

#### Tool Execution
```http
POST /mcp/tools/call
Content-Type: application/json
```

Execute MCP tools with structured parameters.

**Request Body:**
```json
{
  "tool": "search",
  "parameters": {
    "query": "error handling patterns",
    "max_results": 3,
    "include_code_examples": true
  }
}
```

**Response:**
```json
{
  "tool": "search",
  "status": "success",
  "result": {
    "matches": 3,
    "resources": [...]
  },
  "execution_time_ms": 120
}
```

## Error Responses

All APIs use standard HTTP status codes and return structured error responses.

### Error Format
```json
{
  "error": "invalid_query",
  "message": "Search query cannot be empty",
  "code": 400,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Common Status Codes
- `200` - Success
- `400` - Bad Request (invalid parameters)
- `404` - Not Found (document not found)
- `429` - Too Many Requests (rate limited)
- `500` - Internal Server Error
- `503` - Service Unavailable (system overloaded)

## Rate Limiting

Both APIs implement rate limiting to ensure system stability:

- **REST API**: 100 requests/minute per IP
- **MCP API**: 200 requests/minute per IP
- **Search queries**: 10 concurrent searches maximum

Rate limit headers are included in responses:
```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 85
X-RateLimit-Reset: 1642248600
```

## Authentication

Currently, both APIs operate without authentication for local use. For production deployments, consider:

- Reverse proxy with authentication (nginx, Apache)
- API gateway with rate limiting (Kong, Ambassador)
- Network isolation (firewall rules, VPN)

## OpenAPI Specification

Interactive API documentation is available when AgentFS is running:

- **Swagger UI**: `http://localhost:8080/docs`
- **ReDoc**: `http://localhost:8080/redoc`
- **OpenAPI JSON**: `http://localhost:8080/openapi.json`

## SDK Examples

### cURL Examples

**Search with filtering:**
```bash
curl -G "http://localhost:8080/search" \
  --data-urlencode "q=kubernetes deployment" \
  --data-urlencode "sources=devops-docs,k8s-examples" \
  --data-urlencode "limit=10"
```

**Document retrieval:**
```bash
curl "http://localhost:8080/documents/$(echo 'k8s/deployment.yaml' | jq -rR @uri)"
```

### Python Example

```python
import requests

# Search for content
response = requests.get('http://localhost:8080/search', {
    'q': 'machine learning models',
    'limit': 5
})

results = response.json()
for result in results['results']:
    print(f"Found: {result['file_path']} (score: {result['relevance_score']})")
    print(f"Content: {result['chunk_content'][:200]}...")
```

### JavaScript Example

```javascript
// Search with fetch API
async function searchContent(query) {
  const response = await fetch(`http://localhost:8080/search?q=${encodeURIComponent(query)}`);
  const data = await response.json();
  return data.results;
}

// Usage
searchContent('API documentation').then(results => {
  results.forEach(result => {
    console.log(`${result.file_path}: ${result.relevance_score}`);
  });
});
```

## Performance Optimization

### Search Optimization
- Use specific keywords for better relevance
- Combine with source filtering for faster results
- Implement client-side caching for repeated queries

### Pagination
- Use `limit` and `offset` for large result sets
- Maximum recommended page size: 50 results
- Total results count provided for pagination UI

### Caching Strategy
- Search results are cached for 5 minutes
- Document content cached for 30 minutes
- Use ETags for client-side caching
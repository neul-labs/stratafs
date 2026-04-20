# Storage Backends

AgentFS supports multiple storage backends with a unified interface. All storage sources follow a **read-only architecture** - AgentFS never modifies source files or directories.

## Supported Backends

- **Local Filesystem** - Real-time monitoring with immediate indexing
- **Amazon S3** - S3-compatible object storage with configurable endpoints
- **Google Cloud Storage** - Native GCS integration with service account authentication
- **Azure Blob Storage** - Azure blob containers with account key authentication

## Local Storage

Local directories are monitored in real-time with immediate file change detection.

### Configuration
```json
{
  "id": "my-documents",
  "name": "My Documents",
  "type": "local",
  "enabled": true,
  "path": "/home/user/Documents",
  "filters": {
    "include_patterns": ["*"],
    "exclude_patterns": [".git/**", "node_modules/**"],
    "max_file_size": 104857600,
    "ignore_hidden": true
  }
}
```

### Features
- ✅ **Real-time monitoring** - Instant file change detection
- ✅ **Direct processing** - No caching overhead
- ✅ **Cross-platform** - Works on Linux, macOS, Windows
- ✅ **Symbolic links** - Follows symlinks by default

## Amazon S3

S3-compatible object storage with full AWS S3 and S3-compatible endpoint support.

### Configuration
```json
{
  "id": "s3-documents",
  "name": "S3 Documents",
  "type": "s3",
  "enabled": true,
  "path": "my-bucket/documents/",
  "local_cache_dir": "/home/user/.agentfs/cache/s3-documents",
  "credentials": {
    "access_key": "your-access-key",
    "secret_key": "your-secret-key",
    "region": "us-west-2",
    "endpoint": ""  // Optional: for S3-compatible services
  },
  "filters": {
    "include_patterns": ["*.pdf", "*.docx", "*.txt"],
    "max_file_size": 52428800  // 50MB limit for remote files
  }
}
```

### S3-Compatible Services
Works with any S3-compatible storage:
- **AWS S3** - Leave endpoint empty
- **MinIO** - Set endpoint to your MinIO server
- **DigitalOcean Spaces** - Set endpoint to your region
- **Wasabi** - Set endpoint to Wasabi region

### Features
- ✅ **Periodic scanning** - Configurable scan intervals
- ✅ **Change detection** - Timestamp-based modification tracking
- ✅ **Prefix support** - Limit scanning to specific prefixes
- ✅ **Custom endpoints** - Support for S3-compatible services

## Google Cloud Storage

Native GCS integration with service account authentication.

### Configuration
```json
{
  "id": "gcs-documents",
  "name": "GCS Documents",
  "type": "gcs",
  "enabled": true,
  "path": "my-bucket/documents/",
  "local_cache_dir": "/home/user/.agentfs/cache/gcs-documents",
  "credentials": {
    "project_id": "my-project-id",
    "credentials_path": "/path/to/service-account.json"
  },
  "filters": {
    "exclude_patterns": ["*.tmp", "*.log"]
  }
}
```

### Authentication Setup
1. Create a service account in Google Cloud Console
2. Download the JSON credentials file
3. Set `credentials_path` to the file location
4. Ensure the service account has "Storage Object Viewer" permission

### Features
- ✅ **Service account auth** - Secure authentication with JSON key
- ✅ **Project isolation** - Multi-project support
- ✅ **Regional buckets** - Works with any GCS region
- ✅ **IAM integration** - Uses Google Cloud IAM permissions

## Azure Blob Storage

Azure blob containers with account key authentication.

### Configuration
```json
{
  "id": "azure-documents",
  "name": "Azure Documents",
  "type": "azure",
  "enabled": true,
  "path": "my-container/documents/",
  "local_cache_dir": "/home/user/.agentfs/cache/azure-documents",
  "credentials": {
    "account_name": "mystorageaccount",
    "account_key": "your-account-key",
    "container": "my-container"
  }
}
```

### Authentication Setup
1. Get your storage account name and access key from Azure Portal
2. Ensure the storage account has blob read permissions
3. Container must exist before AgentFS can access it

### Features
- ✅ **Account key auth** - Simple authentication with storage keys
- ✅ **Container isolation** - Each source maps to one container
- ✅ **Blob hierarchies** - Supports virtual folder structures
- ✅ **Global regions** - Works with any Azure region

## Read-Only Architecture

All storage backends follow the same read-only principles:

### Source Integrity
- **No modifications** - Source files and directories remain untouched
- **No deletions** - AgentFS never deletes source content
- **No uploads** - No data written back to source storage

### Local Processing
- **Index storage** - All AgentFS metadata in local `.agentfs` directories
- **Cache management** - Remote files cached locally, then cleaned up
- **Database isolation** - Each source gets its own SQLite database

### Workflow Differences

#### Local Sources
1. **Monitor** - Real-time file system events
2. **Process** - Direct file access from original location
3. **Index** - Update search index immediately

#### Remote Sources
1. **Scan** - Periodic polling of remote storage
2. **Download** - Changed files cached locally
3. **Process** - Parse and embed from cache
4. **Cleanup** - Delete cached files after processing
5. **Index** - Update search index

## Performance Considerations

### Local Storage
- **Instant updates** - Changes reflected immediately
- **No bandwidth** - Direct file system access
- **Scalability** - Limited by local disk I/O

### Remote Storage
- **Scan intervals** - Balance between freshness and API costs
- **Bandwidth usage** - Only downloads changed files
- **API limits** - Respects cloud provider rate limits
- **Cache cleanup** - Automatic cleanup prevents disk bloat

### Optimization Tips

#### For Remote Sources
```json
{
  "worker": {
    "scan_interval": "30s"  // Increase for large buckets
  },
  "filters": {
    "max_file_size": 10485760,  // 10MB limit for faster processing
    "exclude_patterns": [
      "*.zip", "*.tar.gz",      // Skip archives
      "*.mp4", "*.avi",         // Skip videos
      "*.jpg", "*.png"          // Skip images
    ]
  }
}
```

#### For Large Datasets
```json
{
  "worker": {
    "count": 8,          // More workers for large datasets
    "batch_size": 20     // Larger batches for efficiency
  },
  "database": {
    "compression_enabled": true,    // Reduce storage overhead
    "maintenance_interval": "12h"   // More frequent cleanup
  }
}
```

## Troubleshooting

### Common Issues

**Credential Errors**
- Verify access keys and permissions
- Check firewall and network connectivity
- Ensure containers/buckets exist

**Performance Issues**
- Adjust scan intervals for large datasets
- Use appropriate file filters
- Monitor cache directory disk usage

**File Access Errors**
- Check file permissions and ownership
- Verify path accessibility
- Review filter patterns for exclusions

### Debug Mode
```bash
# Enable verbose logging
AGENTFS_LOG_LEVEL=debug ./build/agentfs
```

### Health Checks
```bash
# Test storage connectivity
curl http://localhost:8080/health

# View source statistics
curl http://localhost:8080/sources/stats
```
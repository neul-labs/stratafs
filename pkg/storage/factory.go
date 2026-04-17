package storage

import (
	"fmt"
	"agentfs/pkg/config"
	"agentfs/pkg/filesystem"
)

// StorageFactory creates filesystem instances for different storage sources
type StorageFactory struct{}

// NewStorageFactory creates a new storage factory
func NewStorageFactory() *StorageFactory {
	return &StorageFactory{}
}

// CreateFileSystem creates a filesystem implementation for the given storage source
func (sf *StorageFactory) CreateFileSystem(source config.StorageSource) (filesystem.FileSystem, error) {
	switch source.Type {
	case config.StorageTypeLocal:
		return filesystem.NewLocalFileSystem(), nil
	case config.StorageTypeS3:
		return sf.createHybridS3FileSystem(source)
	case config.StorageTypeGCS:
		return sf.createHybridGCSFileSystem(source)
	case config.StorageTypeAzure:
		return sf.createHybridAzureFileSystem(source)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", source.Type)
	}
}

// createHybridS3FileSystem creates a hybrid S3 filesystem with local caching
func (sf *StorageFactory) createHybridS3FileSystem(source config.StorageSource) (filesystem.FileSystem, error) {
	if source.LocalCacheDir == "" {
		return nil, fmt.Errorf("S3 source must have a local cache directory configured")
	}

	// Create the underlying S3 filesystem
	s3fs, err := sf.createS3FileSystem(source)
	if err != nil {
		return nil, err
	}

	// Wrap it with hybrid filesystem for local caching
	return filesystem.NewHybridFileSystem(s3fs, source.LocalCacheDir, source.ID), nil
}

// createHybridGCSFileSystem creates a hybrid GCS filesystem with local caching
func (sf *StorageFactory) createHybridGCSFileSystem(source config.StorageSource) (filesystem.FileSystem, error) {
	if source.LocalCacheDir == "" {
		return nil, fmt.Errorf("GCS source must have a local cache directory configured")
	}

	// Create the underlying GCS filesystem
	gcsfs, err := sf.createGCSFileSystem(source)
	if err != nil {
		return nil, err
	}

	// Wrap it with hybrid filesystem for local caching
	return filesystem.NewHybridFileSystem(gcsfs, source.LocalCacheDir, source.ID), nil
}

// createHybridAzureFileSystem creates a hybrid Azure filesystem with local caching
func (sf *StorageFactory) createHybridAzureFileSystem(source config.StorageSource) (filesystem.FileSystem, error) {
	if source.LocalCacheDir == "" {
		return nil, fmt.Errorf("Azure source must have a local cache directory configured")
	}

	// Create the underlying Azure filesystem
	azurefs, err := sf.createAzureFileSystem(source)
	if err != nil {
		return nil, err
	}

	// Wrap it with hybrid filesystem for local caching
	return filesystem.NewHybridFileSystem(azurefs, source.LocalCacheDir, source.ID), nil
}

// createS3FileSystem creates an S3 filesystem implementation
func (sf *StorageFactory) createS3FileSystem(source config.StorageSource) (filesystem.FileSystem, error) {
	bucket, ok := source.Credentials["bucket"].(string)
	if !ok || bucket == "" {
		return nil, fmt.Errorf("S3 source must specify bucket in credentials")
	}

	region, _ := source.Credentials["region"].(string)
	if region == "" {
		region = "us-east-1" // Default region
	}

	accessKey, _ := source.Credentials["access_key"].(string)
	secretKey, _ := source.Credentials["secret_key"].(string)
	endpoint, _ := source.Credentials["endpoint"].(string)

	return filesystem.NewS3FileSystem(bucket, source.Path, region, accessKey, secretKey, endpoint)
}

// createGCSFileSystem creates a Google Cloud Storage filesystem implementation
func (sf *StorageFactory) createGCSFileSystem(source config.StorageSource) (filesystem.FileSystem, error) {
	bucket, ok := source.Credentials["bucket"].(string)
	if !ok || bucket == "" {
		return nil, fmt.Errorf("GCS source must specify bucket in credentials")
	}

	credentialsPath, _ := source.Credentials["credentials_path"].(string)
	projectID, _ := source.Credentials["project_id"].(string)

	return filesystem.NewGCSFileSystem(bucket, source.Path, projectID, credentialsPath)
}

// createAzureFileSystem creates an Azure Blob Storage filesystem implementation
func (sf *StorageFactory) createAzureFileSystem(source config.StorageSource) (filesystem.FileSystem, error) {
	container, ok := source.Credentials["container"].(string)
	if !ok || container == "" {
		return nil, fmt.Errorf("Azure source must specify container in credentials")
	}

	accountName, _ := source.Credentials["account_name"].(string)
	accountKey, _ := source.Credentials["account_key"].(string)
	connectionString, _ := source.Credentials["connection_string"].(string)

	return filesystem.NewAzureFileSystem(container, source.Path, accountName, accountKey, connectionString)
}

// ValidateSourceCredentials validates that a storage source has the required credentials
func (sf *StorageFactory) ValidateSourceCredentials(source config.StorageSource) error {
	switch source.Type {
	case config.StorageTypeLocal:
		// No credentials needed for local storage
		return nil
	case config.StorageTypeS3:
		return sf.validateS3Credentials(source)
	case config.StorageTypeGCS:
		return sf.validateGCSCredentials(source)
	case config.StorageTypeAzure:
		return sf.validateAzureCredentials(source)
	default:
		return fmt.Errorf("unsupported storage type: %s", source.Type)
	}
}

// validateS3Credentials validates S3 credentials
func (sf *StorageFactory) validateS3Credentials(source config.StorageSource) error {
	bucket, ok := source.Credentials["bucket"].(string)
	if !ok || bucket == "" {
		return fmt.Errorf("S3 source must specify bucket")
	}

	// Access key and secret key are optional if using IAM roles or instance profiles
	accessKey, hasAccessKey := source.Credentials["access_key"].(string)
	secretKey, hasSecretKey := source.Credentials["secret_key"].(string)

	// If one is provided, both must be provided
	if (hasAccessKey && accessKey != "") || (hasSecretKey && secretKey != "") {
		if accessKey == "" || secretKey == "" {
			return fmt.Errorf("S3 source with access_key must also specify secret_key")
		}
	}

	return nil
}

// validateGCSCredentials validates Google Cloud Storage credentials
func (sf *StorageFactory) validateGCSCredentials(source config.StorageSource) error {
	bucket, ok := source.Credentials["bucket"].(string)
	if !ok || bucket == "" {
		return fmt.Errorf("GCS source must specify bucket")
	}

	// Either credentials_path or default application credentials
	credentialsPath, _ := source.Credentials["credentials_path"].(string)
	if credentialsPath == "" {
		// Will use default application credentials (environment variable, metadata server, etc.)
	}

	return nil
}

// validateAzureCredentials validates Azure Blob Storage credentials
func (sf *StorageFactory) validateAzureCredentials(source config.StorageSource) error {
	container, ok := source.Credentials["container"].(string)
	if !ok || container == "" {
		return fmt.Errorf("Azure source must specify container")
	}

	// Must have either connection_string or account_name + account_key
	connectionString, _ := source.Credentials["connection_string"].(string)
	accountName, hasAccountName := source.Credentials["account_name"].(string)
	accountKey, hasAccountKey := source.Credentials["account_key"].(string)

	if connectionString != "" {
		// Connection string is sufficient
		return nil
	}

	if !hasAccountName || accountName == "" || !hasAccountKey || accountKey == "" {
		return fmt.Errorf("Azure source must specify either connection_string or both account_name and account_key")
	}

	return nil
}
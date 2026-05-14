package storage

import (
	"fmt"
	"github.com/neul-labs/stratafs/pkg/config"
	"github.com/neul-labs/stratafs/pkg/filesystem"
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
	case config.StorageTypeSharePoint:
		return sf.createSharePointFileSystem(source)
	case config.StorageTypeGoogleDrive:
		return sf.createGoogleDriveFileSystem(source)
	case config.StorageTypeJira:
		return sf.createJiraFileSystem(source)
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

// createSharePointFileSystem creates a SharePoint filesystem
func (sf *StorageFactory) createSharePointFileSystem(source config.StorageSource) (filesystem.FileSystem, error) {
	if source.LocalCacheDir == "" {
		return nil, fmt.Errorf("SharePoint source must have a local cache directory configured")
	}

	tenantID, _ := source.Credentials["tenant_id"].(string)
	clientID, _ := source.Credentials["client_id"].(string)
	clientSecret, _ := source.Credentials["client_secret"].(string)
	siteURL, _ := source.Credentials["site_url"].(string)
	driveID, _ := source.Credentials["drive_id"].(string)

	if tenantID == "" || clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("SharePoint source must specify tenant_id, client_id, and client_secret")
	}

	config := filesystem.SharePointConfig{
		TenantID:     tenantID,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		SiteURL:      siteURL,
		DriveID:      driveID,
		LocalCache:   source.LocalCacheDir,
	}

	return filesystem.NewSharePointFileSystem(config)
}

// createGoogleDriveFileSystem creates a Google Drive filesystem
func (sf *StorageFactory) createGoogleDriveFileSystem(source config.StorageSource) (filesystem.FileSystem, error) {
	if source.LocalCacheDir == "" {
		return nil, fmt.Errorf("Google Drive source must have a local cache directory configured")
	}

	credentialsFile, _ := source.Credentials["credentials_file"].(string)
	tokenFile, _ := source.Credentials["token_file"].(string)
	folderID, _ := source.Credentials["folder_id"].(string)

	if credentialsFile == "" {
		return nil, fmt.Errorf("Google Drive source must specify credentials_file")
	}

	// Parse export formats if provided
	var exportFormats map[string]string
	if formats, ok := source.Credentials["export_formats"].(map[string]interface{}); ok {
		exportFormats = make(map[string]string)
		for k, v := range formats {
			if s, ok := v.(string); ok {
				exportFormats[k] = s
			}
		}
	}

	config := filesystem.GoogleDriveConfig{
		CredentialsFile: credentialsFile,
		TokenFile:       tokenFile,
		FolderID:        folderID,
		LocalCache:      source.LocalCacheDir,
		ExportFormats:   exportFormats,
	}

	return filesystem.NewGoogleDriveFileSystem(config)
}

// createJiraFileSystem creates a Jira filesystem
func (sf *StorageFactory) createJiraFileSystem(source config.StorageSource) (filesystem.FileSystem, error) {
	if source.LocalCacheDir == "" {
		return nil, fmt.Errorf("Jira source must have a local cache directory configured")
	}

	baseURL, _ := source.Credentials["base_url"].(string)
	email, _ := source.Credentials["email"].(string)
	apiToken, _ := source.Credentials["api_token"].(string)
	jqlFilter, _ := source.Credentials["jql_filter"].(string)

	if baseURL == "" || email == "" || apiToken == "" {
		return nil, fmt.Errorf("Jira source must specify base_url, email, and api_token")
	}

	// Parse projects
	var projects []string
	if p, ok := source.Credentials["projects"].([]interface{}); ok {
		for _, v := range p {
			if s, ok := v.(string); ok {
				projects = append(projects, s)
			}
		}
	}

	if len(projects) == 0 {
		return nil, fmt.Errorf("Jira source must specify at least one project")
	}

	config := filesystem.JiraConfig{
		BaseURL:    baseURL,
		Email:      email,
		APIToken:   apiToken,
		Projects:   projects,
		LocalCache: source.LocalCacheDir,
		JQLFilter:  jqlFilter,
	}

	return filesystem.NewJiraFileSystem(config)
}
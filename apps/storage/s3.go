package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/evo/v2/lib/settings"
)

var (
	s3Client *s3.Client
	bucket   string
	enabled  bool
)

// Config holds S3 configuration
type Config struct {
	Enabled   bool
	Bucket    string
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
}

// Initialize sets up the S3 client
func Initialize() error {
	enabled = settings.Get("S3.ENABLED").Bool()
	if !enabled {
		log.Notice("S3 storage is disabled")
		return nil
	}

	bucket = settings.Get("S3.BUCKET").String()
	endpoint := settings.Get("S3.ENDPOINT").String()
	region := settings.Get("S3.REGION").String()
	accessKey := settings.Get("S3.ACCESS_KEY").String()
	secretKey := settings.Get("S3.SECRET_KEY").String()

	if bucket == "" || endpoint == "" || accessKey == "" || secretKey == "" {
		return fmt.Errorf("S3 configuration incomplete")
	}

	// Add https:// if not present
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	// Create S3 client with custom endpoint (for iDrive E2)
	cfg := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		EndpointResolverWithOptions: aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               endpoint,
					SigningRegion:     region,
					HostnameImmutable: true,
				}, nil
			},
		),
	}

	s3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // Required for S3-compatible services
	})

	log.Notice("S3 storage initialized: bucket=%s, endpoint=%s", bucket, endpoint)
	return nil
}

// IsEnabled returns whether S3 storage is enabled
func IsEnabled() bool {
	return enabled && s3Client != nil
}

// Upload uploads a file to S3
func Upload(ctx context.Context, key string, data []byte, contentType string) error {
	if !IsEnabled() {
		return fmt.Errorf("S3 storage not enabled")
	}

	_, err := s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})

	return err
}

// UploadReader uploads a file from a reader to S3
func UploadReader(ctx context.Context, key string, reader io.Reader, contentType string, contentLength int64) error {
	if !IsEnabled() {
		return fmt.Errorf("S3 storage not enabled")
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
	}

	if contentLength > 0 {
		input.ContentLength = aws.Int64(contentLength)
	}

	_, err := s3Client.PutObject(ctx, input)
	return err
}

// Download downloads a file from S3
func Download(ctx context.Context, key string) ([]byte, string, error) {
	if !IsEnabled() {
		return nil, "", fmt.Errorf("S3 storage not enabled")
	}

	result, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", err
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, "", err
	}

	contentType := ""
	if result.ContentType != nil {
		contentType = *result.ContentType
	}

	return data, contentType, nil
}

// GetReader returns a reader for an S3 object
func GetReader(ctx context.Context, key string) (io.ReadCloser, string, int64, error) {
	if !IsEnabled() {
		return nil, "", 0, fmt.Errorf("S3 storage not enabled")
	}

	result, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", 0, err
	}

	contentType := ""
	if result.ContentType != nil {
		contentType = *result.ContentType
	}

	contentLength := int64(0)
	if result.ContentLength != nil {
		contentLength = *result.ContentLength
	}

	return result.Body, contentType, contentLength, nil
}

// Delete deletes a file from S3
func Delete(ctx context.Context, key string) error {
	if !IsEnabled() {
		return fmt.Errorf("S3 storage not enabled")
	}

	_, err := s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	return err
}

// Exists checks if a file exists in S3
func Exists(ctx context.Context, key string) (bool, error) {
	if !IsEnabled() {
		return false, fmt.Errorf("S3 storage not enabled")
	}

	_, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		// Check if it's a "not found" error
		var notFound *types.NotFound
		if ok := false; !ok {
			// For S3-compatible services, check error message
			if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "404") {
				return false, nil
			}
		}
		if notFound != nil {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// GetObjectInfo returns metadata about an S3 object
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
	ETag         string
}

func GetObjectInfo(ctx context.Context, key string) (*ObjectInfo, error) {
	if !IsEnabled() {
		return nil, fmt.Errorf("S3 storage not enabled")
	}

	result, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	info := &ObjectInfo{
		Key: key,
	}

	if result.ContentLength != nil {
		info.Size = *result.ContentLength
	}
	if result.ContentType != nil {
		info.ContentType = *result.ContentType
	}
	if result.LastModified != nil {
		info.LastModified = *result.LastModified
	}
	if result.ETag != nil {
		info.ETag = *result.ETag
	}

	return info, nil
}

// List lists objects in a prefix
func List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	if !IsEnabled() {
		return nil, fmt.Errorf("S3 storage not enabled")
	}

	result, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, err
	}

	objects := make([]ObjectInfo, 0, len(result.Contents))
	for _, obj := range result.Contents {
		info := ObjectInfo{}
		if obj.Key != nil {
			info.Key = *obj.Key
		}
		if obj.Size != nil {
			info.Size = *obj.Size
		}
		if obj.LastModified != nil {
			info.LastModified = *obj.LastModified
		}
		if obj.ETag != nil {
			info.ETag = *obj.ETag
		}
		objects = append(objects, info)
	}

	return objects, nil
}

// GenerateKey generates a unique key for storing files
func GenerateKey(prefix, filename string) string {
	ext := filepath.Ext(filename)
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s/%d%s", prefix, timestamp, ext)
}

// GetBucket returns the configured bucket name
func GetBucket() string {
	return bucket
}

// GetClient returns the S3 client for advanced operations
func GetClient() *s3.Client {
	return s3Client
}

// DownloadRange downloads a byte range from S3 (for video seeking support)
func DownloadRange(ctx context.Context, key string, rangeHeader string) (io.ReadCloser, string, int64, int64, error) {
	if !IsEnabled() {
		return nil, "", 0, 0, fmt.Errorf("S3 storage not enabled")
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	if rangeHeader != "" {
		input.Range = aws.String(rangeHeader)
	}

	result, err := s3Client.GetObject(ctx, input)
	if err != nil {
		return nil, "", 0, 0, err
	}

	contentType := ""
	if result.ContentType != nil {
		contentType = *result.ContentType
	}

	contentLength := int64(0)
	if result.ContentLength != nil {
		contentLength = *result.ContentLength
	}

	// Get total file size from ContentRange header
	// Format: "bytes 0-999/5000" where 5000 is total size
	totalSize := contentLength
	if result.ContentRange != nil {
		// Parse content range to get total size
		parts := strings.Split(*result.ContentRange, "/")
		if len(parts) == 2 {
			if size, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				totalSize = size
			}
		}
	}

	return result.Body, contentType, contentLength, totalSize, nil
}

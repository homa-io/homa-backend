package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// PresignedURLExpiry is the default expiration time for presigned URLs
const PresignedURLExpiry = 1 * time.Hour

// MultipartUploadExpiry is the default expiration time for multipart uploads
const MultipartUploadExpiry = 24 * time.Hour

// PresignClient wraps the S3 presign client
type PresignClient struct {
	client *s3.PresignClient
}

// NewPresignClient creates a new presign client
func NewPresignClient() *PresignClient {
	if !IsEnabled() {
		return nil
	}
	return &PresignClient{
		client: s3.NewPresignClient(s3Client),
	}
}

// GenerateUploadURL generates a presigned URL for uploading a file
func (p *PresignClient) GenerateUploadURL(ctx context.Context, key, contentType string, expiry time.Duration) (string, error) {
	if expiry == 0 {
		expiry = PresignedURLExpiry
	}

	result, err := p.client.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(expiry))

	if err != nil {
		return "", err
	}

	return result.URL, nil
}

// GenerateDownloadURL generates a presigned URL for downloading a file
func (p *PresignClient) GenerateDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if expiry == 0 {
		expiry = PresignedURLExpiry
	}

	result, err := p.client.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))

	if err != nil {
		return "", err
	}

	return result.URL, nil
}

// MultipartUploadSession holds information about a multipart upload
type MultipartUploadSession struct {
	UploadID string `json:"uploadId"`
	Key      string `json:"key"`
}

// CreateMultipartUpload initiates a multipart upload
func CreateMultipartUpload(ctx context.Context, key, contentType string) (*MultipartUploadSession, error) {
	if !IsEnabled() {
		return nil, fmt.Errorf("S3 storage not enabled")
	}

	result, err := s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, err
	}

	return &MultipartUploadSession{
		UploadID: *result.UploadId,
		Key:      key,
	}, nil
}

// GetPresignedUploadPartURL generates a presigned URL for uploading a part
func GetPresignedUploadPartURL(ctx context.Context, key, uploadID string, partNumber int32) (string, error) {
	if !IsEnabled() {
		return "", fmt.Errorf("S3 storage not enabled")
	}

	presignClient := s3.NewPresignClient(s3Client)

	result, err := presignClient.PresignUploadPart(ctx, &s3.UploadPartInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(key),
		UploadId:   aws.String(uploadID),
		PartNumber: aws.Int32(partNumber),
	}, s3.WithPresignExpires(PresignedURLExpiry))

	if err != nil {
		return "", err
	}

	return result.URL, nil
}

// PartInfo holds information about an uploaded part
type PartInfo struct {
	PartNumber int32  `json:"partNumber"`
	ETag       string `json:"etag"`
}

// CompleteMultipartUpload completes a multipart upload
func CompleteMultipartUpload(ctx context.Context, key, uploadID string, parts []PartInfo) error {
	if !IsEnabled() {
		return fmt.Errorf("S3 storage not enabled")
	}

	// Build the completed parts slice
	var s3Parts []types.CompletedPart
	for _, part := range parts {
		s3Parts = append(s3Parts, types.CompletedPart{
			PartNumber: aws.Int32(part.PartNumber),
			ETag:       aws.String(part.ETag),
		})
	}

	_, err := s3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: s3Parts,
		},
	})

	return err
}

// AbortMultipartUpload aborts a multipart upload
func AbortMultipartUpload(ctx context.Context, key, uploadID string) error {
	if !IsEnabled() {
		return fmt.Errorf("S3 storage not enabled")
	}

	_, err := s3Client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})

	return err
}

// ListParts lists the parts of a multipart upload
func ListParts(ctx context.Context, key, uploadID string) ([]PartInfo, error) {
	if !IsEnabled() {
		return nil, fmt.Errorf("S3 storage not enabled")
	}

	result, err := s3Client.ListParts(ctx, &s3.ListPartsInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	if err != nil {
		return nil, err
	}

	parts := make([]PartInfo, len(result.Parts))
	for i, part := range result.Parts {
		parts[i] = PartInfo{
			PartNumber: *part.PartNumber,
			ETag:       *part.ETag,
		}
	}

	return parts, nil
}

package s3

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/llmariner/job-manager/dispatcher/internal/config"
	laws "github.com/llmariner/common/pkg/aws"
)

// NewClient returns a new S3 client.
func NewClient(ctx context.Context, c config.S3Config) (*Client, error) {
	opts := laws.NewS3ClientOptions{
		EndpointURL: c.EndpointURL,
		Region:      c.Region,
	}
	if ar := c.AssumeRole; ar != nil {
		opts.AssumeRole = &laws.AssumeRole{
			RoleARN:    ar.RoleARN,
			ExternalID: ar.ExternalID,
		}
	}
	svc, err := laws.NewS3Client(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Client{
		svc:    svc,
		bucket: c.Bucket,
	}, nil
}

// Client is a client for S3.
type Client struct {
	svc    *s3.Client
	bucket string
}

// RequestType is the type of the request.
type RequestType string

const (
	// RequestTypeGetObject is the type for getting an object.
	RequestTypeGetObject RequestType = "GetObject"
	// RequestTypePutObject is the type for putting an object.
	RequestTypePutObject RequestType = "PutObject"
)

// GeneratePresignedURL generates a pre-signed URL.
func (c *Client) GeneratePresignedURL(ctx context.Context, key string, expire time.Duration, requestType RequestType) (string, error) {
	presigner := s3.NewPresignClient(c.svc)

	//	var req *request.Request
	switch requestType {
	case RequestTypeGetObject:
		req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(c.bucket),
			Key:    aws.String(key),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = expire
		})
		if err != nil {
			return "", err
		}
		return req.URL, nil
	case RequestTypePutObject:
		req, err := presigner.PresignPutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(c.bucket),
			Key:    aws.String(key),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = expire
		})
		if err != nil {
			return "", err
		}
		return req.URL, nil
	default:
		return "", fmt.Errorf("unknown request type: %s", requestType)
	}
}

// ListObjectsPages returns S3 objects with pagination.
func (c *Client) ListObjectsPages(
	ctx context.Context,
	prefix string,
) (*s3.ListObjectsV2Output, error) {
	return c.svc.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(prefix),
	})
}

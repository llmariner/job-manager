package s3

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	laws "github.com/llmariner/common/pkg/aws"
	"github.com/llmariner/job-manager/dispatcher/internal/config"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// NewClient returns a new S3 client.
func NewClient(ctx context.Context, c config.S3Config) (*Client, error) {
	opts := laws.NewS3ClientOptions{
		EndpointURL:        c.EndpointURL,
		Region:             c.Region,
		InsecureSkipVerify: c.InsecureSkipVerify,
	}
	if ar := c.AssumeRole; ar != nil {
		opts.AssumeRole = &laws.AssumeRole{
			RoleARN:    ar.RoleARN,
			ExternalID: ar.ExternalID,
		}
	}
	if c.SecretFilePath != "" {
		sec, err := os.ReadFile(c.SecretFilePath)
		if err != nil {
			return nil, fmt.Errorf("read S3 secret file: %w", err)
		}
		var secConfig config.AWSSecretConfig
		if err := yaml.Unmarshal(sec, &secConfig); err != nil {
			return nil, fmt.Errorf("unmarshal S3 secret file: %s", err)
		}
		opts.Secret = &laws.Secret{
			AccessKeyID:     secConfig.AccessKeyID,
			SecretAccessKey: secConfig.SecretAccessKey,
		}
	}

	svc, err := laws.NewS3Client(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Client{
		svc: svc,
	}, nil
}

// Client is a client for S3.
type Client struct {
	svc *s3.Client
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
func (c *Client) GeneratePresignedURL(ctx context.Context, bucket, key string, expire time.Duration, requestType RequestType) (string, error) {
	presigner := s3.NewPresignClient(c.svc)

	switch requestType {
	case RequestTypeGetObject:
		req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
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
			Bucket: aws.String(bucket),
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

// GeneratePresignedURLForPost generates a pre-signed URL for a POST request. It allows uploading files
// with given key prefix. For example, when a file 'myfile' is uploaded, the key will be keyPrefix/myfile.
func (c *Client) GeneratePresignedURLForPost(ctx context.Context, bucket, keyPrefix string, expire time.Duration) (*s3.PresignedPostRequest, error) {
	presigner := s3.NewPresignClient(c.svc)
	req, err := presigner.PresignPostObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(keyPrefix + "/${filename}"),
	}, func(opts *s3.PresignPostOptions) {
		opts.Expires = expire
		var conditions []interface{}
		conditions = append(conditions, []interface{}{
			"starts-with",
			"$key",
			keyPrefix + "/",
		})
		opts.Conditions = []interface{}(conditions)
	})
	if err != nil {
		return nil, err
	}

	req.URL, err = reconstructPresignURL(req.URL, bucket)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// reconstructPresignURL reconstructs a pre-signed URL.
// The URL returned by the SDK does not include the bucket name in the URL, which does not work with neither S3 or MinIO.
// This function reconstructs the URL to include the bucket name.
func reconstructPresignURL(origURL, bucket string) (string, error) {
	// construct the URL.
	// If the URL is for S3, the URL should be https://<bucket>.<endpoint>
	// If the URL is fr MinIO, the URL should be http(s)://<endpoint>/<bucket>

	u, err := url.Parse(origURL)
	if err != nil {
		return "", err
	}

	if strings.HasSuffix(u.Host, "amazonaws.com") {
		// S3
		u.Host = fmt.Sprintf("%s.%s", bucket, u.Host)
		return u.String(), nil
	}
	// MinIO
	u.Path = bucket
	return u.String(), nil
}

// ListObjectsPages returns S3 objects with pagination.
func (c *Client) ListObjectsPages(
	ctx context.Context,
	bucket string,
	prefix string,
) (*s3.ListObjectsV2Output, error) {
	return c.svc.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
}

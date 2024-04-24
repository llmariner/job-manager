package s3

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/llm-operator/job-manager/dispatcher/internal/config"
)

// NewClient returns a new S3 client.
func NewClient(c config.S3Config) *Client {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	conf := &aws.Config{
		Endpoint: aws.String(c.EndpointURL),
		Region:   aws.String(c.Region),
		// This is needed as the minio server does not support the virtual host style.
		S3ForcePathStyle: aws.Bool(true),
	}
	return &Client{
		svc:    s3.New(sess, conf),
		bucket: c.Bucket,
	}
}

// Client is a client for S3.
type Client struct {
	svc    *s3.S3
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
func (c *Client) GeneratePresignedURL(key string, expire time.Duration, requestType RequestType) (string, error) {
	var req *request.Request
	switch requestType {
	case RequestTypeGetObject:
		req, _ = c.svc.GetObjectRequest(&s3.GetObjectInput{
			Bucket: aws.String(c.bucket),
			Key:    aws.String(key),
		})
	case RequestTypePutObject:
		req, _ = c.svc.PutObjectRequest(&s3.PutObjectInput{
			Bucket: aws.String(c.bucket),
			Key:    aws.String(key),
		})
	default:
		return "", fmt.Errorf("unknown request type: %s", requestType)
	}
	url, err := req.Presign(expire)
	if err != nil {
		return "", err
	}
	return url, nil
}

// ListObjectsPages returns S3 objects with pagination.
func (c *Client) ListObjectsPages(
	prefix string,
	f func(page *s3.ListObjectsOutput, lastPage bool) bool,
) error {
	return c.svc.ListObjectsPages(&s3.ListObjectsInput{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(prefix),
	}, f)
}

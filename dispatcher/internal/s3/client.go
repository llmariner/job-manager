package s3

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
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
		Region:   aws.String("dummy"),
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

// GeneratePresignedURL generates a pre-signed URL.
//
// TODO(kenji): Limit the presigned URL capability by changing the credentials to be used.
func (c *Client) GeneratePresignedURL(key string, expire time.Duration) (string, error) {
	req, _ := c.svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})

	url, err := req.Presign(expire)
	if err != nil {
		return "", err
	}
	return url, nil
}

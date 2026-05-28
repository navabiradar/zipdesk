package storage

import (
    "bytes"
    "context"
    "fmt"
    "io"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client wraps S3/R2 operations
type Client struct {
    s3         *s3.Client
    bucketName string
    publicURL  string
}

// Config holds storage configuration
type Config struct {
    AccountID  string
    AccessKey  string
    SecretKey  string
    BucketName string
    PublicURL  string
}

// New creates a new storage client
func New(cfg Config) (*Client, error) {
    r2Resolver := aws.EndpointResolverWithOptionsFunc(
        func(service, region string, options ...interface{}) (aws.Endpoint, error) {
            return aws.Endpoint{
                URL: fmt.Sprintf(
                    "https://%s.r2.cloudflarestorage.com",
                    cfg.AccountID,
                ),
            }, nil
        },
    )

    awsCfg, err := config.LoadDefaultConfig(
        context.Background(),
        config.WithEndpointResolverWithOptions(r2Resolver),
        config.WithCredentialsProvider(
            credentials.NewStaticCredentialsProvider(
                cfg.AccessKey,
                cfg.SecretKey,
                "",
            ),
        ),
        config.WithRegion("auto"),
    )
    if err != nil {
        return nil, fmt.Errorf("storage.New: load config: %w", err)
    }

    client := s3.NewFromConfig(awsCfg)

    return &Client{
        s3:         client,
        bucketName: cfg.BucketName,
        publicURL:  cfg.PublicURL,
    }, nil
}

// Upload stores a file and returns its public URL
func (c *Client) Upload(
    ctx context.Context,
    key string,
    data []byte,
    contentType string,
) (string, error) {
    _, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
        Bucket:      aws.String(c.bucketName),
        Key:         aws.String(key),
        Body:        bytes.NewReader(data),
        ContentType: aws.String(contentType),
    })
    if err != nil {
        return "", fmt.Errorf("storage.Upload: %w", err)
    }

    return fmt.Sprintf("%s/%s", c.publicURL, key), nil
}

// UploadStream stores a file from a reader
func (c *Client) UploadStream(
    ctx context.Context,
    key string,
    reader io.Reader,
    contentType string,
) (string, error) {
    data, err := io.ReadAll(reader)
    if err != nil {
        return "", fmt.Errorf("storage.UploadStream: read: %w", err)
    }
    return c.Upload(ctx, key, data, contentType)
}

// Delete removes a file
func (c *Client) Delete(ctx context.Context, key string) error {
    _, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
        Bucket: aws.String(c.bucketName),
        Key:    aws.String(key),
    })
    if err != nil {
        return fmt.Errorf("storage.Delete: %w", err)
    }
    return nil
}

// GetPresignedURL generates a temporary download URL
func (c *Client) GetPresignedURL(
    ctx context.Context,
    key string,
    expiry time.Duration,
) (string, error) {
    presigner := s3.NewPresignClient(c.s3)
    req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(c.bucketName),
        Key:    aws.String(key),
    }, s3.WithPresignExpires(expiry))
    if err != nil {
        return "", fmt.Errorf("storage.GetPresignedURL: %w", err)
    }
    return req.URL, nil
}

// PublicURL returns the public URL for a key
func (c *Client) PublicURL(key string) string {
    return fmt.Sprintf("%s/%s", c.publicURL, key)
}

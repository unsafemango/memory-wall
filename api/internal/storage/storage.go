package storage

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ErrNotFound means the object isn't in the bucket.
var ErrNotFound = errors.New("object not found")

type Config struct {
	// Endpoint is how the API reaches the bucket, e.g. http://minio:9000.
	Endpoint string
	// PublicEndpoint is how browsers reach it, e.g. http://localhost:9000.
	// Presigned URLs are signed for this host. Defaults to Endpoint.
	PublicEndpoint string
	Region         string
	AccessKey      string
	SecretKey      string
	Bucket         string
}

// Storage wraps an S3-compatible bucket (MinIO locally, Cloudflare R2 in
// production).
type Storage struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
}

func New(cfg Config) *Storage {
	if cfg.PublicEndpoint == "" {
		cfg.PublicEndpoint = cfg.Endpoint
	}
	if cfg.Region == "" {
		cfg.Region = "auto"
	}

	newClient := func(endpoint string) *s3.Client {
		return s3.New(s3.Options{
			Region:       cfg.Region,
			Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
			BaseEndpoint: aws.String(endpoint),
			// MinIO serves buckets as a path (host/bucket/key), not a subdomain.
			UsePathStyle: true,
		})
	}

	return &Storage{
		client:    newClient(cfg.Endpoint),
		presigner: s3.NewPresignClient(newClient(cfg.PublicEndpoint)),
		bucket:    cfg.Bucket,
	}
}

// PresignPut returns a URL the browser can PUT one file to. The content type
// and size are part of the signature, so the upload must match them.
func (s *Storage) PresignPut(ctx context.Context, key, contentType string, size int64, expires time.Duration) (string, error) {
	req, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

// PresignGet returns a temporary URL for viewing one file.
func (s *Storage) PresignGet(ctx context.Context, key string, expires time.Duration) (string, error) {
	req, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

// Size returns the object's size in bytes, or ErrNotFound.
func (s *Storage) Size(ctx context.Context, key string) (int64, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return aws.ToInt64(out.ContentLength), nil
}

// Download streams an object's contents. The caller closes the reader.
func (s *Storage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

// Upload stores contents under key.
func (s *Storage) Upload(ctx context.Context, key, contentType string, body io.Reader) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
		Body:        body,
	})
	return err
}

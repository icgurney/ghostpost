package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type TigrisStorage struct {
	client *s3.Client
	bucket string
}

func NewTigrisStorage(client *s3.Client, bucket string) *TigrisStorage {
	return &TigrisStorage{
		client: client,
		bucket: bucket,
	}
}

func NewTigrisClient(ctx context.Context) (*s3.Client, error) {
	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't load config: %w", err)
	}

	// Create Tigris service client
	client := s3.NewFromConfig(sdkConfig, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("https://fly.storage.tigris.dev")
		o.Region = "auto"
	})

	return client, nil
}

func (s *TigrisStorage) SaveEmail(ctx context.Context, id string, body io.Reader) error {
	key := fmt.Sprintf("emails/%s/%s.eml", time.Now().Format("2006/01/02"), id)

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   body,
	})
	return err
}

func (s *TigrisStorage) GetEmails(ctx context.Context, user string) ([]types.Object, error) {
	emails, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(fmt.Sprintf("emails/%s", user)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list emails: %w", err)
	}
	return emails.Contents, nil
}

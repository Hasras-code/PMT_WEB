package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"strings"
	"time"

	"github.com/Hasras-code/PMT_WEB.git/internal/apperror"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

type R2Config struct {
	Endpoint, Region       string
	AccessKeyID, SecretKey string
	PrivateBucket          string
	PublicBucket           string
	PublicBaseURL          string
}

type R2 struct {
	client        *s3.Client
	presign       *s3.PresignClient
	privateBucket string
	publicBucket  string
	publicBaseURL string
}

func (r *R2) Check(ctx context.Context) error {
	for _, bucket := range []string{r.privateBucket, r.publicBucket} {
		if _, err := r.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)}); err != nil {
			return fmt.Errorf("object storage bucket is unavailable")
		}
	}
	return nil
}

func OpenR2(ctx context.Context, cfg R2Config) (*R2, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("configure R2: %w", err)
	}
	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(cfg.Endpoint)
		options.UsePathStyle = true
	})
	return &R2{
		client:        client,
		presign:       s3.NewPresignClient(client),
		privateBucket: cfg.PrivateBucket,
		publicBucket:  cfg.PublicBucket,
		publicBaseURL: cfg.PublicBaseURL,
	}, nil
}

func (r *R2) bucket(class string) (string, error) {
	switch class {
	case ClassPrivate:
		return r.privateBucket, nil
	case ClassPublic:
		return r.publicBucket, nil
	default:
		return "", apperror.ErrInvalid
	}
}

func (r *R2) CreateUploadURL(ctx context.Context, object Object, ttl time.Duration) (UploadAuthorization, error) {
	if !validObjectKey(object.Key) || object.Size < 1 || object.Size > 50<<20 || !safeExtension(strings.ToLower(pathExtension(object.Name)), object.MIME) {
		return UploadAuthorization{}, apperror.ErrInvalid
	}
	bucket, err := r.bucket(object.Class)
	if err != nil {
		return UploadAuthorization{}, err
	}
	result, err := r.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(object.Key),
		ContentLength: aws.Int64(object.Size),
		ContentType:   aws.String(object.MIME),
	}, func(options *s3.PresignOptions) { options.Expires = ttl })
	if err != nil {
		return UploadAuthorization{}, fmt.Errorf("create R2 upload URL: %w", err)
	}
	return UploadAuthorization{URL: result.URL, Headers: map[string]string{"Content-Type": object.MIME}, ExpiresIn: int64(ttl.Seconds())}, nil
}

func (r *R2) Inspect(ctx context.Context, object Object) (Metadata, error) {
	bucket, err := r.bucket(object.Class)
	if err != nil {
		return Metadata{}, err
	}
	head, err := r.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(bucket), Key: aws.String(object.Key)})
	if err != nil {
		return Metadata{}, storageError(err)
	}
	if head.ContentLength == nil || *head.ContentLength != object.Size || head.ContentType == nil || strings.Split(*head.ContentType, ";")[0] != object.MIME {
		return Metadata{}, apperror.ErrInvalid
	}
	get, err := r.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(object.Key), Range: aws.String("bytes=0-1048575")})
	if err != nil {
		return Metadata{}, storageError(err)
	}
	defer get.Body.Close()
	return inspect(get.Body, object.MIME, object.Size)
}

func (r *R2) CreateDownloadURL(ctx context.Context, object Object, ttl time.Duration) (string, error) {
	bucket, err := r.bucket(object.Class)
	if err != nil {
		return "", err
	}
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": object.Name})
	result, err := r.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket:                     aws.String(bucket),
		Key:                        aws.String(object.Key),
		ResponseContentDisposition: aws.String(disposition),
		ResponseContentType:        aws.String(object.MIME),
	}, func(options *s3.PresignOptions) { options.Expires = ttl })
	if err != nil {
		return "", fmt.Errorf("create R2 download URL: %w", err)
	}
	return result.URL, nil
}

func (r *R2) PublicURL(key string) (string, error) { return joinPublicURL(r.publicBaseURL, key) }

func (r *R2) Delete(ctx context.Context, object Object) error {
	bucket, err := r.bucket(object.Class)
	if err != nil {
		return err
	}
	_, err = r.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: aws.String(object.Key)})
	return storageError(err)
}

func (r *R2) Put(ctx context.Context, object Object, reader io.Reader) (Metadata, error) {
	bucket, err := r.bucket(object.Class)
	if err != nil {
		return Metadata{}, err
	}
	_, err = r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(object.Key),
		Body:          reader,
		ContentLength: aws.Int64(object.Size),
		ContentType:   aws.String(object.MIME),
	})
	if err != nil {
		return Metadata{}, fmt.Errorf("upload R2 object: %w", err)
	}
	return r.Inspect(ctx, object)
}

func storageError(err error) error {
	if err == nil {
		return nil
	}
	var api smithy.APIError
	if errors.As(err, &api) && (api.ErrorCode() == "NotFound" || api.ErrorCode() == "NoSuchKey") {
		return apperror.ErrNotFound
	}
	return fmt.Errorf("object storage operation failed")
}

func pathExtension(name string) string {
	i := strings.LastIndexByte(name, '.')
	if i < 0 {
		return ""
	}
	return name[i:]
}

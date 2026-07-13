package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOService struct {
	client     *minio.Client
	bucketName string
}

func NewMinIOService(endpoint, accessKey, secretKey, bucketName string, useSSL bool) (*MinIOService, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		if err := client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
		log.Printf("MinIO bucket '%s' created successfully", bucketName)
	}

	log.Println("MinIO service initialized successfully")
	return &MinIOService{
		client:     client,
		bucketName: bucketName,
	}, nil
}

func (m *MinIOService) UploadVerificationVideo(ctx context.Context, maidID uuid.UUID, file io.Reader, fileSize int64) (string, error) {
	objectName := fmt.Sprintf("verification/videos/%s_%d.mp4", maidID.String(), time.Now().Unix())
	
	_, err := m.client.PutObject(ctx, m.bucketName, objectName, file, fileSize, minio.PutObjectOptions{
		ContentType: "video/mp4",
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload video: %w", err)
	}

	log.Printf("Uploaded verification video: %s", objectName)
	return objectName, nil
}

func (m *MinIOService) UploadIDPhoto(ctx context.Context, maidID uuid.UUID, file io.Reader, fileSize int64, contentType string) (string, error) {
	ext := ".jpg"
	if contentType == "image/png" {
		ext = ".png"
	}
	
	objectName := fmt.Sprintf("verification/ids/%s_%d%s", maidID.String(), time.Now().Unix(), ext)
	
	_, err := m.client.PutObject(ctx, m.bucketName, objectName, file, fileSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload ID photo: %w", err)
	}

	log.Printf("Uploaded ID photo: %s", objectName)
	return objectName, nil
}

func (m *MinIOService) UploadProfilePhoto(ctx context.Context, userID uuid.UUID, file io.Reader, fileSize int64, contentType string) (string, error) {
	ext := filepath.Ext(contentType)
	if ext == "" {
		ext = ".jpg"
	}
	
	objectName := fmt.Sprintf("profiles/original/%s%s", userID.String(), ext)
	
	_, err := m.client.PutObject(ctx, m.bucketName, objectName, file, fileSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload profile photo: %w", err)
	}

	log.Printf("Uploaded profile photo: %s", objectName)
	return objectName, nil
}

func (m *MinIOService) UploadImage(ctx context.Context, folder string, ownerID uuid.UUID, file io.Reader, fileSize int64, contentType string) (string, error) {
	ext := ".jpg"
	if contentType == "image/png" {
		ext = ".png"
	}

	objectName := fmt.Sprintf("%s/%s_%d%s", folder, ownerID.String(), time.Now().UnixNano(), ext)

	_, err := m.client.PutObject(ctx, m.bucketName, objectName, file, fileSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload image to %s: %w", folder, err)
	}

	log.Printf("Uploaded image: %s", objectName)
	return objectName, nil
}

func (m *MinIOService) GetPresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	url, err := m.client.PresignedGetObject(ctx, m.bucketName, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return url.String(), nil
}

func (m *MinIOService) GetPublicURL(objectName string) string {
	return fmt.Sprintf("http://%s/%s/%s", m.client.EndpointURL().Host, m.bucketName, objectName)
}
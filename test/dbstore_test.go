package main

import (
	"testing"

	"context"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func TestDBListing(t *testing.T) {
	// Define the MinIO server configuration for access from the host
	endpoint := "localhost:9000"
	accessKeyID := "user"
	secretAccessKey := "mmusty8432"
	useSSL := false

	// Initialize the MinIO client
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		t.Errorf("Cannot init client")
	}

	// Define the bucket name
	bucketName := "qldb"

	// Create a context
	ctx := context.Background()

	// List all objects in the bucket
	objectCh := minioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			t.Errorf("Cannot access key listing")
		}

		t.Logf("Object Key: %s\n", object.Key)

	}
}

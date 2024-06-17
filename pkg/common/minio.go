package common

import (
	"context"
	"log/slog"

	"github.com/minio/minio-go/v7"
)

func CreateMinIOBucketIfNotExists(client *minio.Client, bucketName string) error {
	ctx := context.Background()

	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return err
	}

	if !exists {
		slog.Info("Creating bucket", "name", bucketName)
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			// The bucket might already exist at this stage if another component created it concurrently.
			// For example, the server might have attempted to create it at the same time as the agent.
			if err.(minio.ErrorResponse).Code == "BucketAlreadyOwnedByYou" {
				slog.Info("Failed to create bucket because it already exists", "name", bucketName)
				return nil
			} else {
				return err
			}
		} else {
			slog.Info("Bucket created successfully", "name", bucketName)
		}
	} else {
		slog.Info("Bucket already exists", "name", bucketName)
	}

	return nil
}

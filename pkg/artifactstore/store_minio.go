package artifactstore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"mrvacommander/pkg/common"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOArtifactStore struct {
	client *minio.Client
}

func NewMinIOArtifactStore(endpoint, id, secret string) (*MinIOArtifactStore, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(id, secret, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}

	slog.Info("Connected to MinIO artifact store server")

	// Create "results" bucket
	if err := common.CreateMinIOBucketIfNotExists(minioClient, AF_VAL_BUCKET_RESULTS); err != nil {
		return nil, fmt.Errorf("could not create results bucket: %v", err)
	}

	// Create "packs" bucket
	if err := common.CreateMinIOBucketIfNotExists(minioClient, AF_VAL_BUCKET_PACKS); err != nil {
		return nil, fmt.Errorf("could not create packs bucket: %v", err)
	}

	return &MinIOArtifactStore{
		client: minioClient,
	}, nil
}

func (store *MinIOArtifactStore) GetQueryPack(location ArtifactLocation) ([]byte, error) {
	return store.getArtifact(location)
}

func (store *MinIOArtifactStore) SaveQueryPack(jobId int, data []byte) (ArtifactLocation, error) {
	return store.saveArtifact(AF_VAL_BUCKET_PACKS, deriveKeyFromSessionId(jobId), data, "application/gzip")
}

func (store *MinIOArtifactStore) GetResult(location ArtifactLocation) ([]byte, error) {
	return store.getArtifact(location)
}

func (store *MinIOArtifactStore) GetResultSize(location ArtifactLocation) (int, error) {
	// bucket := location.Data[AF_KEY_BUCKET]
	// key := location.Data[AF_KEY_KEY]
	bucket := location.Bucket
	key := location.Key

	objectInfo, err := store.client.StatObject(context.Background(), bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return 0, err
	}

	if objectInfo.Size > math.MaxInt32 {
		return 0, fmt.Errorf("object size %d exceeds max int size", objectInfo.Size)
	}

	return int(objectInfo.Size), nil
}

func (store *MinIOArtifactStore) SaveResult(jobSpec common.JobSpec, data []byte) (ArtifactLocation, error) {
	return store.saveArtifact(AF_VAL_BUCKET_RESULTS, deriveKeyFromJobSpec(jobSpec), data, "application/zip")
}

func (store *MinIOArtifactStore) getArtifact(location ArtifactLocation) ([]byte, error) {
	bucket := location.Bucket
	key := location.Key
	// bucket := location.Data[AF_KEY_BUCKET]
	// key := location.Data[AF_KEY_KEY]

	object, err := store.client.GetObject(context.Background(), bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer object.Close()

	data, err := io.ReadAll(object)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (store *MinIOArtifactStore) saveArtifact(bucket, key string, data []byte,
	contentType string) (ArtifactLocation, error) {
	_, err := store.client.PutObject(context.Background(), bucket, key,
		bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
			ContentType: contentType,
		})
	if err != nil {
		return ArtifactLocation{}, err
	}

	// XX: static types
	location := ArtifactLocation{
		Bucket: bucket,
		Key:    key,
		// Data: map[string]string{
		// 	AF_KEY_BUCKET: bucket,
		// 	AF_KEY_KEY:    key,
		// },
	}

	return location, nil
}

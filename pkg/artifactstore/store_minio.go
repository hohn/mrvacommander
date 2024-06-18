package artifactstore

import (
	"bytes"
	"context"
	"fmt"
	"io"

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

	return &MinIOArtifactStore{
		client: minioClient,
	}, nil
}

func (store *MinIOArtifactStore) GetQueryPack(location ArtifactLocation) ([]byte, error) {
	return store.getArtifact(location)
}

func (store *MinIOArtifactStore) SaveQueryPack(sessionID int, data []byte) (ArtifactLocation, error) {
	return store.saveArtifact("packs", sessionID, data)
}

func (store *MinIOArtifactStore) GetResult(location ArtifactLocation) ([]byte, error) {
	return store.getArtifact(location)
}

func (store *MinIOArtifactStore) SaveResult(sessionID int, data []byte) (ArtifactLocation, error) {
	return store.saveArtifact("results", sessionID, data)
}

func (store *MinIOArtifactStore) getArtifact(location ArtifactLocation) ([]byte, error) {
	bucket := location.data["bucket"]
	key := location.data["key"]

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

func (store *MinIOArtifactStore) saveArtifact(bucket string, sessionID int, data []byte) (ArtifactLocation, error) {
	key := fmt.Sprintf("%d.tgz", sessionID)
	_, err := store.client.PutObject(context.Background(), bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/gzip",
	})
	if err != nil {
		return ArtifactLocation{}, err
	}

	location := ArtifactLocation{
		data: map[string]string{
			"bucket": bucket,
			"key":    key,
		},
	}

	return location, nil
}

package qldbstore

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mrvacommander/pkg/artifactstore"
	"mrvacommander/pkg/common"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const bucketName = "qldb"

type MinIOCodeQLDatabaseStore struct {
	client     *minio.Client
	bucketName string
}

func NewMinIOCodeQLDatabaseStore(endpoint, id, secret string) (*MinIOCodeQLDatabaseStore, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(id, secret, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}

	slog.Info("Connected to MinIO CodeQL database store server")

	err = common.CreateMinIOBucketIfNotExists(minioClient, bucketName)
	if err != nil {
		return nil, fmt.Errorf("could not create bucket: %v", err)
	}

	return &MinIOCodeQLDatabaseStore{
		client:     minioClient,
		bucketName: bucketName,
	}, nil
}

func (store *MinIOCodeQLDatabaseStore) FindAvailableDBs(analysisReposRequested []common.NameWithOwner) (
	notFoundRepos []common.NameWithOwner,
	foundRepos *map[common.NameWithOwner]CodeQLDatabaseLocation) {

	foundReposMap := make(map[common.NameWithOwner]CodeQLDatabaseLocation)
	for _, repo := range analysisReposRequested {
		location, err := store.GetDatabaseLocationByNWO(repo)
		if err != nil {
			notFoundRepos = append(notFoundRepos, repo)
		} else {
			foundReposMap[repo] = location
		}
	}

	return notFoundRepos, &foundReposMap
}

func (store *MinIOCodeQLDatabaseStore) GetDatabase(location CodeQLDatabaseLocation) ([]byte, error) {
	bucket := location.data[artifactstore.AF_KEY_BUCKET]
	key := location.data[artifactstore.AF_KEY_KEY]

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

func (store *MinIOCodeQLDatabaseStore) GetDatabaseLocationByNWO(nwo common.NameWithOwner) (CodeQLDatabaseLocation, error) {
	objectName := fmt.Sprintf("%s$%s.zip", nwo.Owner, nwo.Repo)

	// Check if the object exists
	_, err := store.client.StatObject(context.Background(), store.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return CodeQLDatabaseLocation{}, fmt.Errorf("database not found for %s", nwo)
		}
		return CodeQLDatabaseLocation{}, err
	}

	location := CodeQLDatabaseLocation{
		data: map[string]string{
			"bucket": store.bucketName,
			"key":    objectName,
		},
	}

	return location, nil
}

package qldbstore

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mrvacommander/pkg/common"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// XX: static types: split by type?
// Restrict the keys / values and centralize the common ones here
const (
	QL_DB_BUCKETNAME = "qldb"
)

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

	err = common.CreateMinIOBucketIfNotExists(minioClient, QL_DB_BUCKETNAME)
	if err != nil {
		return nil, fmt.Errorf("could not create bucket: %v", err)
	}

	return &MinIOCodeQLDatabaseStore{
		client:     minioClient,
		bucketName: QL_DB_BUCKETNAME,
	}, nil
}

func (store *MinIOCodeQLDatabaseStore) FindAvailableDBs(analysisReposRequested []common.NameWithOwner) (
	notFoundRepos []common.NameWithOwner,
	foundRepos []common.NameWithOwner) {

	for _, repo := range analysisReposRequested {
		status := store.haveDatabase(repo)
		if status {
			foundRepos = append(foundRepos, repo)
		} else {
			notFoundRepos = append(notFoundRepos, repo)
		}
	}

	return notFoundRepos, foundRepos
}

func (store *MinIOCodeQLDatabaseStore) GetDatabase(location common.NameWithOwner) ([]byte, error) {
	key := fmt.Sprintf("%s$%s.zip", location.Owner, location.Repo)
	object, err := store.client.GetObject(context.Background(),
		store.bucketName,
		key,
		minio.GetObjectOptions{})
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

func (store *MinIOCodeQLDatabaseStore) haveDatabase(location common.NameWithOwner) bool {
	objectName := fmt.Sprintf("%s$%s.zip", location.Owner, location.Repo)

	// Check if the object exists
	_, err := store.client.StatObject(context.Background(),
		store.bucketName,
		objectName,
		minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			slog.Info("No database found for", location)
			return false
		}
		slog.Info("General database error while checking for", location)
		return false
	}
	return true
}

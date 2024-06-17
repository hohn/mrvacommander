package qldbstore

import (
	"mrvacommander/pkg/common"
)

type DBLocation struct {
	Prefix string
	File   string
}

type Storage interface {
	FindAvailableDBs(analysisReposRequested []common.NameWithOwner) (not_found_repos []common.NameWithOwner,
		analysisRepos *map[common.NameWithOwner]DBLocation)
}

type Visibles struct{}

type StorageQLDB struct{}

func NewStore(v *Visibles) (Storage, error) {
	s := StorageQLDB{}

	return &s, nil
}

func (s *StorageQLDB) FindAvailableDBs(analysisReposRequested []common.NameWithOwner) (
	not_found_repos []common.NameWithOwner,
	analysisRepos *map[common.NameWithOwner]DBLocation) {
	// TODO implement
	return nil, nil
}

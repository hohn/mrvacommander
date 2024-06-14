package qpstore

import (
	"fmt"
	"log/slog"
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/qldbstore"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DBmutex sync.Mutex
)

type StorageVisibles struct{}

type StorageContainer struct {
	// Database version of StorageSingle
	RequestID int
	DB        *gorm.DB
	modules   *StorageVisibles
}

type DBSpec struct {
	Host     string
	Port     int
	User     string
	Password string
	DBname   string
}

func (s *StorageContainer) SetupDB() error {
	// TODO set up query pack storage
	return nil
}

func (s *StorageContainer) LoadState() error {
	// TODO load the state
	return nil
}

func (s *StorageContainer) hasTables() bool {
	// TODO query to check for tables
	return false
}

func (s *StorageContainer) NextID() int {
	// TODO update via db
	return 12345
}

func (s *StorageContainer) SaveQueryPack(tgz []byte, sessionID int) (storagePath string, error error) {
	// TODO save and return path
	return "todo:no-path-yet", nil
}

func (s *StorageContainer) FindAvailableDBs(analysisReposRequested []common.NameWithOwner) (notFoundRepos []common.NameWithOwner, analysisRepos *map[common.NameWithOwner]qldbstore.DBLocation) {
	// TODO  s.FindAvailableDBs() via postgres
	analysisRepos = &map[common.NameWithOwner]qldbstore.DBLocation{}
	notFoundRepos = []common.NameWithOwner{}

	return notFoundRepos, analysisRepos
}

func (s *StorageContainer) Setup(v *StorageVisibles) {
	s.modules = v
}

func NewStore(startingID int) (Storage, error) {
	// TODO drop the startingID

	db, err := ConnectDB(DBSpec{
		Host:     "postgres",
		Port:     5432,
		User:     "exampleuser",
		Password: "examplepass",
		DBname:   "querypack_db",
	})
	if err != nil {
		return nil, err
	}

	s := StorageContainer{RequestID: startingID, DB: db}
	if err := s.SetupDB(); err != nil {
		return nil, err
	}

	if err = s.loadState(); err != nil {
		return nil, err
	}

	return &s, nil
}

func NewStorageContainer(startingID int) (*StorageContainer, error) {

	db, err := ConnectDB(DBSpec{
		Host:     "postgres",
		Port:     5432,
		User:     "exampleuser",
		Password: "examplepass",
		DBname:   "server_db",
	})
	if err != nil {
		return nil, err
	}

	s := StorageContainer{RequestID: startingID, DB: db}
	if err := s.SetupDB(); err != nil {
		return nil, err
	}

	if err = s.loadState(); err != nil {
		return nil, err
	}

	return &s, nil
}

func ConnectDB(s DBSpec) (*gorm.DB, error) {
	// Open the database connection
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		s.Host, s.Port, s.User, s.Password, s.DBname)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("Error connecting to the database", "err", err)
		return nil, err
	}
	return db, nil
}

func (s *StorageContainer) loadState() error {
	// TODO load the state
	return nil
}

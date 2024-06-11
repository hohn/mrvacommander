package storage

import (
	"fmt"
	"log/slog"
	"mrvacommander/pkg/common"
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DBmutex sync.Mutex
)

func (s *StorageContainer) NextID() int {
	// TODO update via db
	return 12345
}

func (s *StorageContainer) SaveQueryPack(tgz []byte, sessionID int) (storagePath string, error error) {
	// TODO save and return path
	return "todo:no-path-yet", nil
}

func (s *StorageContainer) FindAvailableDBs(analysisReposRequested []common.OwnerRepo) (notFoundRepos []common.OwnerRepo, analysisRepos *map[common.OwnerRepo]DBLocation) {
	// TODO  s.FindAvailableDBs() via postgres
	analysisRepos = &map[common.OwnerRepo]DBLocation{}
	notFoundRepos = []common.OwnerRepo{}

	return notFoundRepos, analysisRepos
}

func (s *StorageContainer) Setup(v *Visibles) {
	s.modules = v
}

func NewQLDBStore() (*StorageContainer, error) {
	// TODO set up qldb_db
	return nil, nil
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

func (s *StorageContainer) SetupDB() error {
	msg := "Failed to initialize database "

	if err := s.DB.AutoMigrate(&DBInfo{}); err != nil {
		slog.Error(msg, "table", "dbinfo")
		return err
	}
	if err := s.DB.AutoMigrate(&DBJobs{}); err != nil {
		slog.Error(msg, "table", "dbjobs")
		return err
	}
	if err := s.DB.AutoMigrate(&DBResult{}); err != nil {
		slog.Error(msg, "table", "dbresult")
		return err
	}
	if err := s.DB.AutoMigrate(&DBStatus{}); err != nil {
		slog.Error(msg, "table", "dbstatus")
		return err
	}

	return nil
}

func (s *StorageContainer) loadState() error {
	// TODO load the state
	return nil
}

func (s *StorageContainer) hasTables() bool {
	// TODO sql query to check for tables
	return false
}

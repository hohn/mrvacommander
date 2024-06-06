package main

import (
	"testing"

	"mrvacommander/pkg/storage"
)

func TestSetupDB(t *testing.T) {

	db, err := storage.ConnectDB(storage.DBSpec{
		Host:     "localhost",
		Port:     5432,
		User:     "exampleuser",
		Password: "examplepass",
		DBname:   "exampledb",
	})

	if err != nil {
		t.Errorf("Cannot connect to db")
	}

	// Check and set up the database
	s := storage.StorageContainer{RequestID: 12345, DB: db}
	if err := s.SetupDB(); err != nil {
		t.Errorf("Cannot set up db")
	}

}

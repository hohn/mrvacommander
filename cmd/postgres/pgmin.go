package main

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"mrvacommander/pkg/common"
)

// TODO migrate this to test/
// TODO add a reader test
// Minimal gorm example that takes a go struct, creates a postgres table,
// and writes the struct to the table.
func main() {
	// Set up the database connection string
	dsn := "host=postgres user=exampleuser dbname=exampledb sslmode=disable password=examplepass"

	// Open the database connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema: create the 'owner_repo' table from the struct
	err = db.AutoMigrate(&common.NameWithOwner{})
	if err != nil {
		panic("failed to migrate database")
	}

	// Create an entry in the database
	db.Create(&common.NameWithOwner{Owner: "foo", Repo: "foo/bar"})
}

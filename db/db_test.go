package db_test

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/tomasstrnad1997/mines/db"
)

func createTempDB(t *testing.T) (string, error) {
	t.Helper()
	// Create a temporary file for the SQLite database
	tempFile, err := os.CreateTemp("", "*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	// Close the temp file
	tempFile.Close()
	t.Cleanup(func() {

		if err := os.Remove(tempFile.Name()); err != nil {
			fmt.Printf("failed to delete temp DB file %v\n", err)
		}
	})

	// Open the SQLite database
	database, err := sql.Open("sqlite3", tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to open db file: %v", err)
	}

	if err = db.InitializeTables(database); err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	// Return the temp file path for cleanup later
	return tempFile.Name(), nil
}

func TestDBcreation(t *testing.T) {
	filename, err := createTempDB(t)
	if err != nil {
		t.Fatalf("Failed to create temp db: %v", err)
	}
	os.Setenv("DB_PATH", filename)
	store, err := db.InitStore()
	if err != nil {
		t.Fatalf("Failed to create Store: %v", err)
	}
	defer store.DB.Close()
	name := "John"
	pwHash := "NOT HASHED"
	err = store.CreatePlayer(name, pwHash)
	if err != nil {
		t.Fatalf("Failed to store player in db: %v", err)
	}
}

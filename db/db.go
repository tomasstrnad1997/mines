package db

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"

	"github.com/tomasstrnad1997/mines/db/store"
	"github.com/tomasstrnad1997/mines/mines"
	"github.com/tomasstrnad1997/mines/players"
)

//go:embed sqlc/schema.sql
var ddl string

type SQLStore struct {
	Q   store.Queries
	DB  *sql.DB
	ctx context.Context
}

func InitializeTables(db *sql.DB) error {
	_, err := db.Exec(ddl)
	return err
}

func (store *SQLStore) InitializeTables() error {
	err := InitializeTables(store.DB)
	if err != nil {
		return err
	}
	return store.InsertGamemodes()
}

func InitStore() (*SQLStore, error) {
	path := os.Getenv("DB_PATH")
	if path == "" {
		return nil, fmt.Errorf("DB_PATH not set in environment")
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	// Need to ping the database to check if the file could be opened
	if err = db.Ping(); err != nil {
		return nil, err
	}
	ctx := context.Background()

	store := &SQLStore{Q: *store.New(db), ctx: ctx, DB: db}
	return store, nil
}

func (s *SQLStore) CreatePlayer(name, hash string) error {
	params := store.CreatePlayerParams{Username: name, PasswordHash: hash}
	_, err := s.Q.CreatePlayer(s.ctx, params)
	return err
}

func (s *SQLStore) FindPlayerByName(name string) (*players.Player, error) {
	p, err := s.Q.GetPlayerByUsername(s.ctx, name)
	if err != nil {
		return nil, err
	}
	plr := &players.Player{ID: uint32(p.ID), Name: p.Username, PasswordHash: p.PasswordHash}
	return plr, nil
}

func (s *SQLStore) InsertGamemodes() error {
	for id, name := range mines.GameModeNames {
		params := store.InsertGamemodesParams{ID: int64(id), Name: name}
		if err := s.Q.InsertGamemodes(s.ctx, params); err != nil {
			return err
		}
	}
	return nil
}

package db

import (
	"context"
	"database/sql"
	_ "embed"
	_ "github.com/mattn/go-sqlite3"
	"fmt"
	"os"

	"github.com/tomasstrnad1997/mines/db/store"
	"github.com/tomasstrnad1997/mines/players"
)

//go:embed sqlc/schema.sql
var ddl string

type SQLStore struct {
	Q   store.Queries
	DB *sql.DB
	ctx context.Context
}

func CreateTables(db *sql.DB) error {
	_, err := db.Exec(ddl)
	return err
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

package db

import (
	"context"
	"database/sql"

	"github.com/tomasstrnad1997/mines/db/store"
	"github.com/tomasstrnad1997/mines/players"
)


type SQLStore struct {
	Q store.Queries
	ctx context.Context 
}

func CreateStore(DB *sql.DB) *SQLStore{
	return &SQLStore{Q:*store.New(DB)}
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

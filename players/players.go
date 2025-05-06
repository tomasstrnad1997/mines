package players

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	Store PlayerStore
}

var ErrInvalidCredentials = errors.New("invalid username or password")
func (s *Service) Register(username, password string) error {
	passwordHash, err := hashPassword(password)
	if err != nil {
		return err
	}
	return s.Store.CreatePlayer(username, passwordHash)
}

func (s *Service) Login(username, password string) (*Player, error){
	player, err := s.Store.FindPlayerByName(username)
	if err != nil {
		return nil, err
	}
	if !checkPasswordHash(password, player.PasswordHash) {
		return nil, ErrInvalidCredentials
	}
	return player, nil
}

func hashPassword(password string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return "", err
    }
    return string(hash), nil
}

func checkPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}

package players

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	Store PlayerStore
}

type AuthToken struct {
	PlayerID  uint32
	Expiry    int64
	ServerID  uint32
	Nonce     [16]byte
	Signature [32]byte
}

type PlayerInfo struct {
	ID   uint32
	Name string
}

const AuthTokenLength = 4 + 4 + 8 + 16 + 32

var (
	ErrTokenExpired     = errors.New("token has expired")
	ErrInvalidSignature = errors.New("invalid signature")
	ErrInvalidFormat    = errors.New("invalid token format")

	ErrInvalidCredentials = errors.New("invalid username or password")
)

func (s *Service) Register(username, password string) error {
	passwordHash, err := hashPassword(password)
	if err != nil {
		return err
	}
	return s.Store.CreatePlayer(username, passwordHash)
}

func (s *Service) Login(username, password string) (*PlayerInfo, error) {
	player, err := s.Store.FindPlayerByName(username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if !checkPasswordHash(password, player.PasswordHash) {
		return nil, ErrInvalidCredentials
	}
	return &PlayerInfo{Name: player.Name, ID: player.ID}, nil
}

func (s *Service) FindPlayerByName(name string) (*Player, error) {
	return s.Store.FindPlayerByName(name)
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

func GenerateAuthToken(player *Player, serverID uint32, secret []byte, ttl time.Duration) (AuthToken, error) {
	expiration := time.Now().Add(ttl).Unix()
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return AuthToken{}, err
	}
	toSign := constructSignatureData(player.ID, serverID, nonce, expiration)
	signature, err := calculateSignature(toSign, secret)
	if err != nil {
		return AuthToken{}, err
	}
	return AuthToken{
		PlayerID:  player.ID,
		ServerID:  serverID,
		Expiry:    expiration,
		Nonce:     nonce,
		Signature: [32]byte(signature),
	}, nil
}

func constructSignatureData(playerID, serverID uint32, nonce [16]byte, expiration int64) []byte {
	// playerID + serverID + expiration + nonce
	data := make([]byte, 4+4+8+16)
	binary.BigEndian.PutUint32(data[0:4], playerID)
	binary.BigEndian.PutUint32(data[4:8], serverID)
	binary.BigEndian.PutUint64(data[8:16], uint64(expiration))
	copy(data[16:], nonce[:])
	return data

}

func ValidateAuthToken(token AuthToken, secret []byte) (bool, error) {
	if time.Now().Unix() > token.Expiry {
		return false, ErrTokenExpired
	}
	toVerify := constructSignatureData(token.PlayerID, token.ServerID, token.Nonce, token.Expiry)
	expectedSignature, err := calculateSignature(toVerify, secret)
	if err != nil {
		return false, ErrInvalidFormat
	}
	if !hmac.Equal(token.Signature[:], expectedSignature) {
		return false, ErrInvalidSignature
	}
	return true, nil
}

func calculateSignature(data []byte, key []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, key)
	if _, err := mac.Write(data); err != nil {
		return nil, err
	}
	return mac.Sum(nil), nil

}

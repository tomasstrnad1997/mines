package players_test

import (
	"errors"
	"testing"
	"time"

	"github.com/tomasstrnad1997/mines/players"
)

func TestTokenValidation(t *testing.T) {
	secret := []byte("SECRET TOKEN")
	player := players.Player{
		ID: 1235,
	}
	token, err := players.GenerateAuthToken(&player, secret, time.Minute*10)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	success, err := players.ValidateAuthToken(token, secret)
	if !success {
		t.Fatalf("Verification failed: %v", err)
	}
}

func TestTokenExpiration(t *testing.T) {
	secret := []byte("SECRET TOKEN")
	player := players.Player{
		ID: 1235,
	}
	token, err := players.GenerateAuthToken(&player, secret, time.Minute*-1)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	success, err := players.ValidateAuthToken(token, secret)
	if success {
		t.Fatalf("Expired token was validated as success: %v", err)
	}
	if !errors.Is(err, players.ErrTokenExpired) {
		t.Fatalf("Expected ErrTOkenExpired, got: %v", err)
	}
}

func TestTokenModification(t *testing.T) {
	secret := []byte("SECRET TOKEN")
	player := players.Player{
		ID: 1235,
	}
	token, err := players.GenerateAuthToken(&player, secret, time.Minute*-1)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	token.Expiry = time.Now().Add(time.Hour).Unix()
	success, err := players.ValidateAuthToken(token, secret)
	if success {
		t.Fatalf("Modified token was validated as success: %v", err)
	}
	if !errors.Is(err, players.ErrInvalidSignature) {
		t.Fatalf("Expected ErrInvalidSignature, got: %v", err)
	}
}

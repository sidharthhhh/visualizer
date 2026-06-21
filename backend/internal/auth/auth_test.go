package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("testpassword")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Fatal("HashPassword returned empty hash")
	}

	valid, err := VerifyPassword("testpassword", hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}

	if !valid {
		t.Fatal("VerifyPassword returned false for correct password")
	}

	valid, err = VerifyPassword("wrongpassword", hash)
	if err != nil {
		t.Fatalf("VerifyPassword failed: %v", err)
	}

	if valid {
		t.Fatal("VerifyPassword returned true for wrong password")
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	secret := "test-secret"
	userID := uuid.New()

	token, err := GenerateToken(userID, uuid.Nil, "", TokenAccess, secret, 15*time.Minute)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("GenerateToken returned empty token")
	}

	claims, err := ValidateToken(token, secret, TokenAccess)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.UserID != userID {
		t.Fatalf("Expected UserID %v, got %v", userID, claims.UserID)
	}

	if claims.Type != TokenAccess {
		t.Fatalf("Expected type %v, got %v", TokenAccess, claims.Type)
	}
}

func TestValidateTokenWrongType(t *testing.T) {
	secret := "test-secret"
	userID := uuid.New()

	token, err := GenerateToken(userID, uuid.Nil, "", TokenAccess, secret, 15*time.Minute)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = ValidateToken(token, secret, TokenRefresh)
	if err == nil {
		t.Fatal("ValidateToken should fail for wrong token type")
	}
}

func TestValidateTokenExpired(t *testing.T) {
	secret := "test-secret"
	userID := uuid.New()

	token, err := GenerateToken(userID, uuid.Nil, "", TokenAccess, secret, -1*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = ValidateToken(token, secret, TokenAccess)
	if err == nil {
		t.Fatal("ValidateToken should fail for expired token")
	}
}

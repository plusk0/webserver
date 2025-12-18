package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMakeJWT(t *testing.T) {
	// Setup
	userID := uuid.New()
	tokenSecret := "test-secret"

	// Test valid JWT creation
	tokenString, err := MakeJWT(userID, tokenSecret)
	assert.NoError(t, err, "MakeJWT should not return an error")
	assert.NotEmpty(t, tokenString, "Token string should not be empty")

	// Optionally, verify the token is valid
	_, err = ValidateJWT(tokenString, tokenSecret)
	assert.NoError(t, err, "ValidateJWT should not return an error for a valid token")
}

func TestValidateJWT(t *testing.T) {
	// Setup
	userID := uuid.New()
	tokenSecret := "test-secret"
	expiresIn := time.Hour

	// Generate a valid token
	tokenString, err := MakeJWT(userID, tokenSecret)
	assert.NoError(t, err, "MakeJWT should not return an error")

	// Test valid token
	validatedUserID, err := ValidateJWT(tokenString, tokenSecret)
	assert.NoError(t, err, "ValidateJWT should not return an error for a valid token")
	assert.Equal(t, userID, validatedUserID, "Validated userID should match the original userID")

	// Test invalid token (wrong secret)
	_, err = ValidateJWT(tokenString, "wrong-secret")
	assert.Error(t, err, "ValidateJWT should return an error for an invalid token")

	// Test expired token - depracated
	expiredToken, err := MakeJWT(userID, tokenSecret)
	assert.NoError(t, err, "MakeJWT should not return an error")
	_, err = ValidateJWT(expiredToken, tokenSecret)
	assert.Error(t, err, "ValidateJWT should return an error for an expired token")

	// Test malformed token
	_, err = ValidateJWT("malformed.token.string", tokenSecret)
	assert.Error(t, err, "ValidateJWT should return an error for a malformed token")

	// Test invalid UUID in Subject claim
	claims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
		Issuer:    "chirpy",
		Subject:   "not-a-uuid",
		ID:        userID.String(),
		Audience:  []string{"somebody_else"},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	invalidUUIDToken, err := token.SignedString([]byte(tokenSecret)) // Fix: Convert tokenSecret to []byte
	assert.NoError(t, err, "Signing should not return an error")
	_, err = ValidateJWT(invalidUUIDToken, tokenSecret)
	assert.Error(t, err, "ValidateJWT should return an error for an invalid UUID in Subject claim")
}

func TestGetBearerToken(t *testing.T) {
	// Test case 1: Valid Authorization header with Bearer token and single space
	validHeaders1 := http.Header{}
	validHeaders1.Set("Authorization", "Bearer TOKEN_STRING")
	token, err := GetBearerToken(validHeaders1)
	assert.NoError(t, err, "Should not return an error for valid Authorization header")
	assert.Equal(t, "TOKEN_STRING", token, "Should return the token string without Bearer prefix and whitespace")
	// Test case 2: Missing Authorization header
	emptyHeaders := http.Header{}
	_, err = GetBearerToken(emptyHeaders)
	assert.Error(t, err, "Should return an error when Authorization header is missing")
	assert.Equal(t, "no auth found", err.Error(), "Should return the correct error message")
}

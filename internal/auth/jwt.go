package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var (
	// ErrInvalidToken is returned when token validation fails
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken is returned when token has expired
	ErrExpiredToken = errors.New("token has expired")
)

// Claims represents JWT token claims
type Claims struct {
	UserID      int64    `json:"user_id"`
	Login       string   `json:"login"`
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	AvatarURL   string   `json:"avatar_url"`
	GitHubToken string   `json:"github_token"` // Encrypted
	Roles       []string `json:"roles,omitempty"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT token generation and validation
type JWTManager struct {
	secretKey       []byte
	sessionDuration time.Duration
	encryptionKey   []byte
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey string, sessionDurationHours int) (*JWTManager, error) {
	if secretKey == "" {
		return nil, errors.New("secret key is required")
	}

	// Use first 32 bytes of secret for AES-256 encryption
	encKey := make([]byte, 32)
	copy(encKey, []byte(secretKey))

	return &JWTManager{
		secretKey:       []byte(secretKey),
		sessionDuration: time.Duration(sessionDurationHours) * time.Hour,
		encryptionKey:   encKey,
	}, nil
}

// GenerateToken creates a new JWT token for the user
func (m *JWTManager) GenerateToken(user *GitHubUser, githubToken string) (string, error) {
	// Encrypt GitHub token before storing in JWT
	encryptedToken, err := m.encryptToken(githubToken)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt token: %w", err)
	}

	now := time.Now()
	claims := &Claims{
		UserID:      user.ID,
		Login:       user.Login,
		Name:        user.Name,
		Email:       user.Email,
		AvatarURL:   user.AvatarURL,
		GitHubToken: encryptedToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.sessionDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "github-migrator",
			Subject:   user.Login,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates and parses a JWT token
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// DecryptGitHubToken decrypts the GitHub token from claims
func (m *JWTManager) DecryptGitHubToken(encryptedToken string) (string, error) {
	return m.decryptToken(encryptedToken)
}

// encryptToken encrypts a token using AES-GCM
func (m *JWTManager) encryptToken(plaintext string) (string, error) {
	block, err := aes.NewCipher(m.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptToken decrypts a token using AES-GCM
// DecryptToken decrypts an encrypted token (public method for use in middleware)
func (m *JWTManager) DecryptToken(ciphertext string) (string, error) {
	return m.decryptToken(ciphertext)
}

func (m *JWTManager) decryptToken(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(m.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// RefreshToken creates a new token with extended expiration
func (m *JWTManager) RefreshToken(claims *Claims) (string, error) {
	// Create new claims with extended expiration
	now := time.Now()
	newClaims := &Claims{
		UserID:      claims.UserID,
		Login:       claims.Login,
		Name:        claims.Name,
		Email:       claims.Email,
		AvatarURL:   claims.AvatarURL,
		GitHubToken: claims.GitHubToken,
		Roles:       claims.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.sessionDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "github-migrator",
			Subject:   claims.Login,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	tokenString, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

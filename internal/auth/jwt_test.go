package auth

import (
	"testing"
	"time"
)

func TestNewJWTManager(t *testing.T) {
	tests := []struct {
		name        string
		secretKey   string
		duration    int
		expectError bool
	}{
		{
			name:        "valid secret key",
			secretKey:   "test-secret-key-32-characters-long",
			duration:    24,
			expectError: false,
		},
		{
			name:        "empty secret key",
			secretKey:   "",
			duration:    24,
			expectError: true,
		},
		{
			name:        "short secret key",
			secretKey:   "short",
			duration:    24,
			expectError: false, // Still works, just not secure
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewJWTManager(tt.secretKey, tt.duration)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if manager == nil {
				t.Error("Expected manager but got nil")
			}
		})
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	manager, err := NewJWTManager("test-secret-key-at-least-32-chars-long", 24)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	user := &GitHubUser{
		ID:        12345,
		Login:     "testuser",
		Name:      "Test User",
		Email:     "test@example.com",
		AvatarURL: "https://avatars.githubusercontent.com/u/12345",
	}

	githubToken := "ghp_test1234567890"

	// Generate token
	token, err := manager.GenerateToken(user, githubToken)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	if token == "" {
		t.Error("Generated token is empty")
	}

	// Validate token
	claims, err := manager.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	// Check claims
	if claims.UserID != user.ID {
		t.Errorf("UserID mismatch: got %d, want %d", claims.UserID, user.ID)
	}
	if claims.Login != user.Login {
		t.Errorf("Login mismatch: got %s, want %s", claims.Login, user.Login)
	}
	if claims.Name != user.Name {
		t.Errorf("Name mismatch: got %s, want %s", claims.Name, user.Name)
	}
	if claims.Email != user.Email {
		t.Errorf("Email mismatch: got %s, want %s", claims.Email, user.Email)
	}
	if claims.GitHubToken == "" {
		t.Error("GitHub token is empty in claims")
	}

	// Decrypt GitHub token
	decryptedToken, err := manager.DecryptGitHubToken(claims.GitHubToken)
	if err != nil {
		t.Fatalf("Failed to decrypt GitHub token: %v", err)
	}
	if decryptedToken != githubToken {
		t.Errorf("GitHub token mismatch: got %s, want %s", decryptedToken, githubToken)
	}
}

func TestValidateInvalidToken(t *testing.T) {
	manager, err := NewJWTManager("test-secret-key-at-least-32-chars-long", 24)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "malformed token",
			token: "not-a-valid-jwt",
		},
		{
			name:  "random string",
			token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.ValidateToken(tt.token)
			if err == nil {
				t.Error("Expected error for invalid token but got none")
			}
		})
	}
}

func TestValidateTokenWithDifferentSecret(t *testing.T) {
	manager1, _ := NewJWTManager("secret-key-one-at-least-32-chars", 24)
	manager2, _ := NewJWTManager("secret-key-two-at-least-32-chars", 24)

	user := &GitHubUser{
		ID:    12345,
		Login: "testuser",
	}

	token, err := manager1.GenerateToken(user, "github-token")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Try to validate with different secret
	_, err = manager2.ValidateToken(token)
	if err == nil {
		t.Error("Expected error when validating token with different secret")
	}
}

func TestRefreshToken(t *testing.T) {
	manager, err := NewJWTManager("test-secret-key-at-least-32-chars-long", 1)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	user := &GitHubUser{
		ID:    12345,
		Login: "testuser",
	}

	// Generate original token
	originalToken, err := manager.GenerateToken(user, "github-token")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Get claims
	originalClaims, err := manager.ValidateToken(originalToken)
	if err != nil {
		t.Fatalf("Failed to validate original token: %v", err)
	}

	// Small delay to ensure different issued time
	time.Sleep(10 * time.Millisecond)

	// Refresh token
	newToken, err := manager.RefreshToken(originalClaims)
	if err != nil {
		t.Fatalf("Failed to refresh token: %v", err)
	}

	// Validate new token
	newClaims, err := manager.ValidateToken(newToken)
	if err != nil {
		t.Fatalf("Failed to validate refreshed token: %v", err)
	}

	// Check that user info is preserved
	if newClaims.UserID != originalClaims.UserID {
		t.Error("UserID changed after refresh")
	}
	if newClaims.Login != originalClaims.Login {
		t.Error("Login changed after refresh")
	}

	// Check that expiry time is in the future
	if newClaims.ExpiresAt.Before(time.Now()) {
		t.Error("Refreshed token should have future expiry time")
	}

	// Check that issued time is recent (within last second)
	if time.Since(newClaims.IssuedAt.Time) > time.Second {
		t.Error("IssuedAt should be recent after refresh")
	}
}

func TestTokenEncryption(t *testing.T) {
	manager, err := NewJWTManager("test-secret-key-at-least-32-chars-long", 24)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	plaintext := "sensitive-github-token-12345"

	// Encrypt
	encrypted, err := manager.encryptToken(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt token: %v", err)
	}
	if encrypted == plaintext {
		t.Error("Encrypted token should not equal plaintext")
	}

	// Decrypt
	decrypted, err := manager.decryptToken(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt token: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("Decrypted token mismatch: got %s, want %s", decrypted, plaintext)
	}
}

func TestTokenEncryptionUniqueness(t *testing.T) {
	manager, _ := NewJWTManager("test-secret-key-at-least-32-chars-long", 24)

	plaintext := "same-token-12345"

	// Encrypt twice
	encrypted1, _ := manager.encryptToken(plaintext)
	encrypted2, _ := manager.encryptToken(plaintext)

	// Should be different due to random nonce
	if encrypted1 == encrypted2 {
		t.Error("Two encryptions of the same plaintext should produce different ciphertexts")
	}

	// But both should decrypt to the same plaintext
	decrypted1, _ := manager.decryptToken(encrypted1)
	decrypted2, _ := manager.decryptToken(encrypted2)

	if decrypted1 != plaintext || decrypted2 != plaintext {
		t.Error("Both encryptions should decrypt to original plaintext")
	}
}

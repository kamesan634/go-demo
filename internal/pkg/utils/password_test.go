package utils

import (
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "securepassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Error("Expected hash to be non-empty")
	}

	if hash == password {
		t.Error("Hash should not equal plain password")
	}

	// bcrypt hash should start with $2a$ or $2b$
	if !strings.HasPrefix(hash, "$2") {
		t.Error("Expected bcrypt hash prefix")
	}
}

func TestHashPassword_TooShort(t *testing.T) {
	_, err := HashPassword("short")
	if err != ErrPasswordTooShort {
		t.Errorf("Expected ErrPasswordTooShort, got %v", err)
	}
}

func TestHashPassword_TooLong(t *testing.T) {
	// bcrypt has a max length of 72 bytes
	longPassword := strings.Repeat("a", 73)

	_, err := HashPassword(longPassword)
	if err != ErrPasswordTooLong {
		t.Errorf("Expected ErrPasswordTooLong, got %v", err)
	}
}

func TestCheckPassword_Correct(t *testing.T) {
	password := "securepassword123"

	hash, _ := HashPassword(password)

	if !CheckPassword(password, hash) {
		t.Error("Expected password check to pass")
	}
}

func TestCheckPassword_Wrong(t *testing.T) {
	password := "securepassword123"
	wrongPassword := "wrongpassword"

	hash, _ := HashPassword(password)

	if CheckPassword(wrongPassword, hash) {
		t.Error("Expected password check to fail")
	}
}

func TestCheckPassword_InvalidHash(t *testing.T) {
	if CheckPassword("password", "not-a-valid-hash") {
		t.Error("Expected password check to fail with invalid hash")
	}
}

func TestValidatePassword_Valid(t *testing.T) {
	validPasswords := []string{
		"password123",
		"12345678",
		"a" + strings.Repeat("b", 70), // 71 chars, just under limit
	}

	for _, pw := range validPasswords {
		if err := ValidatePassword(pw); err != nil {
			t.Errorf("Expected password '%s' to be valid, got error: %v", pw, err)
		}
	}
}

func TestValidatePassword_TooShort(t *testing.T) {
	shortPasswords := []string{
		"",
		"a",
		"1234567", // 7 chars
	}

	for _, pw := range shortPasswords {
		err := ValidatePassword(pw)
		if err != ErrPasswordTooShort {
			t.Errorf("Expected ErrPasswordTooShort for password '%s', got %v", pw, err)
		}
	}
}

func TestValidatePassword_TooLong(t *testing.T) {
	longPassword := strings.Repeat("a", 73)

	err := ValidatePassword(longPassword)
	if err != ErrPasswordTooLong {
		t.Errorf("Expected ErrPasswordTooLong, got %v", err)
	}
}

func TestHashPassword_Uniqueness(t *testing.T) {
	password := "samepassword123"

	hash1, _ := HashPassword(password)
	hash2, _ := HashPassword(password)

	// Same password should produce different hashes (due to salt)
	if hash1 == hash2 {
		t.Error("Expected different hashes for same password (due to salt)")
	}

	// But both hashes should validate against the password
	if !CheckPassword(password, hash1) {
		t.Error("Expected hash1 to validate")
	}
	if !CheckPassword(password, hash2) {
		t.Error("Expected hash2 to validate")
	}
}

func TestHashPassword_EdgeCases(t *testing.T) {
	testCases := []struct {
		password string
		valid    bool
	}{
		{"12345678", true},                  // Exactly 8 chars
		{strings.Repeat("a", 72), true},     // Exactly 72 chars (max)
		{"日本語パスワード!", true},                   // Unicode characters
		{"password with spaces", true},      // Spaces
		{"!@#$%^&*()[]{}|", true},           // Special chars
		{"\t\n\r password", true},           // Whitespace chars
	}

	for _, tc := range testCases {
		hash, err := HashPassword(tc.password)
		if tc.valid {
			if err != nil {
				t.Errorf("Expected password '%s' to hash successfully, got error: %v", tc.password, err)
			}
			if !CheckPassword(tc.password, hash) {
				t.Errorf("Expected password '%s' to verify against its hash", tc.password)
			}
		}
	}
}

func BenchmarkHashPassword(b *testing.B) {
	password := "benchmarkpassword123"

	for i := 0; i < b.N; i++ {
		HashPassword(password)
	}
}

func BenchmarkCheckPassword(b *testing.B) {
	password := "benchmarkpassword123"
	hash, _ := HashPassword(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CheckPassword(password, hash)
	}
}

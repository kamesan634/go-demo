package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Claims represents JWT claims
type Claims struct {
	UserID   string    `json:"user_id"`
	Username string    `json:"username"`
	Type     TokenType `json:"type"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT operations
type JWTManager struct {
	secretKey       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	issuer          string
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey string, accessTTL, refreshTTL time.Duration, issuer string) *JWTManager {
	return &JWTManager{
		secretKey:       []byte(secretKey),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
		issuer:          issuer,
	}
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// GenerateTokenPair generates both access and refresh tokens
func (m *JWTManager) GenerateTokenPair(userID, username string) (*TokenPair, error) {
	accessToken, accessExp, err := m.generateToken(userID, username, AccessToken, m.accessTokenTTL)
	if err != nil {
		return nil, err
	}

	refreshToken, _, err := m.generateToken(userID, username, RefreshToken, m.refreshTokenTTL)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExp,
	}, nil
}

// GenerateAccessToken generates only an access token
func (m *JWTManager) GenerateAccessToken(userID, username string) (string, time.Time, error) {
	return m.generateToken(userID, username, AccessToken, m.accessTokenTTL)
}

// GenerateRefreshToken generates only a refresh token
func (m *JWTManager) GenerateRefreshToken(userID, username string) (string, time.Time, error) {
	return m.generateToken(userID, username, RefreshToken, m.refreshTokenTTL)
}

func (m *JWTManager) generateToken(userID, username string, tokenType TokenType, ttl time.Duration) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(ttl)

	claims := &Claims{
		UserID:   userID,
		Username: username,
		Type:     tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", time.Time{}, err
	}

	return signedToken, expiresAt, nil
}

// ValidateToken validates a token and returns the claims
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
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

// ValidateAccessToken validates an access token
func (m *JWTManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.Type != AccessToken {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token
func (m *JWTManager) ValidateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.Type != RefreshToken {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetTokenID extracts the token ID from claims
func (m *JWTManager) GetTokenID(tokenString string) (string, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.ID, nil
}

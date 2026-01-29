package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-demo/chat/internal/pkg/utils"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func createTestJWTManager() *utils.JWTManager {
	return utils.NewJWTManager(
		"test-secret-key",
		15*time.Minute,
		7*24*time.Hour,
		"test-issuer",
	)
}

func TestAuth_NoToken(t *testing.T) {
	router := setupTestRouter()
	jwtManager := createTestJWTManager()

	router.GET("/protected", Auth(jwtManager), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuth_InvalidFormat(t *testing.T) {
	router := setupTestRouter()
	jwtManager := createTestJWTManager()

	router.GET("/protected", Auth(jwtManager), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "InvalidFormat token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuth_EmptyToken(t *testing.T) {
	router := setupTestRouter()
	jwtManager := createTestJWTManager()

	router.GET("/protected", Auth(jwtManager), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	router := setupTestRouter()
	jwtManager := createTestJWTManager()

	router.GET("/protected", Auth(jwtManager), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuth_ValidToken(t *testing.T) {
	router := setupTestRouter()
	jwtManager := createTestJWTManager()

	// Generate a valid token
	tokenPair, err := jwtManager.GenerateTokenPair("user-123", "testuser")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	router.GET("/protected", Auth(jwtManager), func(c *gin.Context) {
		userID := GetUserID(c)
		username := GetUsername(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id":  userID,
			"username": username,
		})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestOptionalAuth_NoToken(t *testing.T) {
	router := setupTestRouter()
	jwtManager := createTestJWTManager()

	router.GET("/optional", OptionalAuth(jwtManager), func(c *gin.Context) {
		if IsAuthenticated(c) {
			c.JSON(http.StatusOK, gin.H{"authenticated": true})
		} else {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
		}
	})

	req := httptest.NewRequest("GET", "/optional", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestOptionalAuth_ValidToken(t *testing.T) {
	router := setupTestRouter()
	jwtManager := createTestJWTManager()

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "testuser")

	router.GET("/optional", OptionalAuth(jwtManager), func(c *gin.Context) {
		if IsAuthenticated(c) {
			c.JSON(http.StatusOK, gin.H{
				"authenticated": true,
				"user_id":       GetUserID(c),
			})
		} else {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
		}
	})

	req := httptest.NewRequest("GET", "/optional", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestOptionalAuth_InvalidToken(t *testing.T) {
	router := setupTestRouter()
	jwtManager := createTestJWTManager()

	router.GET("/optional", OptionalAuth(jwtManager), func(c *gin.Context) {
		if IsAuthenticated(c) {
			c.JSON(http.StatusOK, gin.H{"authenticated": true})
		} else {
			c.JSON(http.StatusOK, gin.H{"authenticated": false})
		}
	})

	req := httptest.NewRequest("GET", "/optional", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should still return 200, just not authenticated
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetUserID(t *testing.T) {
	router := setupTestRouter()
	jwtManager := createTestJWTManager()

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "testuser")

	var capturedUserID string

	router.GET("/test", Auth(jwtManager), func(c *gin.Context) {
		capturedUserID = GetUserID(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if capturedUserID != "user-123" {
		t.Errorf("Expected user_id 'user-123', got '%s'", capturedUserID)
	}
}

func TestGetUsername(t *testing.T) {
	router := setupTestRouter()
	jwtManager := createTestJWTManager()

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "testuser")

	var capturedUsername string

	router.GET("/test", Auth(jwtManager), func(c *gin.Context) {
		capturedUsername = GetUsername(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if capturedUsername != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", capturedUsername)
	}
}

func TestIsAuthenticated(t *testing.T) {
	router := setupTestRouter()
	jwtManager := createTestJWTManager()

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "testuser")

	var isAuth bool

	router.GET("/test", Auth(jwtManager), func(c *gin.Context) {
		isAuth = IsAuthenticated(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if !isAuth {
		t.Error("Expected IsAuthenticated to return true")
	}
}

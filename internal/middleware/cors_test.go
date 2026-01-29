package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORS_Headers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS())

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// With AllowCredentials=true (default), it returns the actual origin instead of *
	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin == "" {
		t.Error("Expected Access-Control-Allow-Origin to be set")
	}

	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("Expected Access-Control-Allow-Credentials 'true', got '%s'", w.Header().Get("Access-Control-Allow-Credentials"))
	}

	allowHeaders := w.Header().Get("Access-Control-Allow-Headers")
	if allowHeaders == "" {
		t.Error("Expected Access-Control-Allow-Headers to be set")
	}

	allowMethods := w.Header().Get("Access-Control-Allow-Methods")
	if allowMethods == "" {
		t.Error("Expected Access-Control-Allow-Methods to be set")
	}
}

func TestCORS_PreflightRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS())

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Preflight should return 204 No Content
	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204 for preflight, got %d", w.Code)
	}
}

func TestCORS_AllowMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS())

	router.POST("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	router.PUT("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	router.DELETE("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	methods := []string{"POST", "PUT", "DELETE"}

	for _, method := range methods {
		req := httptest.NewRequest(method, "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for %s, got %d", method, w.Code)
		}

		// Check that CORS headers are present
		origin := w.Header().Get("Access-Control-Allow-Origin")
		if origin == "" {
			t.Errorf("Expected CORS origin header for %s", method)
		}
	}
}

func TestCORS_NoOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS())

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// Request without Origin header (same-origin request)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should still work
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCORS_CustomConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	config := &CORSConfig{
		AllowOrigins:     []string{"http://example.com"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: false,
	}
	router.Use(CORSWithConfig(config))

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// Allowed origin
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	origin := w.Header().Get("Access-Control-Allow-Origin")
	if origin != "http://example.com" {
		t.Errorf("Expected origin 'http://example.com', got '%s'", origin)
	}
}

func TestCORS_AllowMethodsHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS())

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	allowMethods := w.Header().Get("Access-Control-Allow-Methods")
	expectedMethods := []string{"GET", "POST", "PUT", "DELETE"}

	for _, method := range expectedMethods {
		if !strings.Contains(allowMethods, method) {
			t.Errorf("Expected Allow-Methods to contain %s, got '%s'", method, allowMethods)
		}
	}
}

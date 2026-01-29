package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestID_SetsHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	requestID := w.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("Expected X-Request-ID header to be set")
	}
}

func TestRequestID_UniquePerRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	requestIDs := make(map[string]bool)

	// Make multiple requests
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		requestID := w.Header().Get("X-Request-ID")
		if requestID == "" {
			t.Error("Expected X-Request-ID header to be set")
			continue
		}

		if requestIDs[requestID] {
			t.Errorf("Duplicate request ID: %s", requestID)
		}
		requestIDs[requestID] = true
	}
}

func TestRequestID_UsesProvidedID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	providedID := "custom-request-id-123"

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", providedID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	requestID := w.Header().Get("X-Request-ID")
	if requestID != providedID {
		t.Errorf("Expected request ID '%s', got '%s'", providedID, requestID)
	}
}

func TestRequestID_AvailableInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())

	var capturedRequestID string

	router.GET("/test", func(c *gin.Context) {
		capturedRequestID = GetRequestID(c)
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	responseRequestID := w.Header().Get("X-Request-ID")
	if capturedRequestID != responseRequestID {
		t.Errorf("Context request ID '%s' doesn't match response header '%s'", capturedRequestID, responseRequestID)
	}
}

func TestRequestID_ValidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	requestID := w.Header().Get("X-Request-ID")

	// UUID should be 36 characters (8-4-4-4-12 format)
	if len(requestID) != 36 {
		t.Errorf("Expected UUID format (36 chars), got '%s' (%d chars)", requestID, len(requestID))
	}
}

func TestGetRequestID_NoMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// No RequestID middleware

	var capturedRequestID string

	router.GET("/test", func(c *gin.Context) {
		capturedRequestID = GetRequestID(c)
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return empty string
	if capturedRequestID != "" {
		t.Errorf("Expected empty string when middleware not used, got '%s'", capturedRequestID)
	}
}

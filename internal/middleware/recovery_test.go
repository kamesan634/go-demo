package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func createRecoveryTestLogger() (*zap.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	encoderConfig := zap.NewProductionEncoderConfig()
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	core := zapcore.NewCore(encoder, zapcore.AddSync(buf), zapcore.DebugLevel)
	logger := zap.New(core)
	return logger, buf
}

func TestRecovery_RecoversPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := createRecoveryTestLogger()

	router := gin.New()
	router.Use(Recovery(logger))

	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()

	// Should not panic
	router.ServeHTTP(w, req)

	// Should return 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestRecovery_ReturnsJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := createRecoveryTestLogger()

	router := gin.New()
	router.Use(Recovery(logger))

	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Check response is JSON
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("Expected JSON content type, got '%s'", contentType)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Check response has error field with message
	errorField, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Error("Expected response to have 'error' field")
		return
	}
	if errorField["message"] == nil {
		t.Error("Expected error to have 'message' field")
	}
}

func TestRecovery_LogsPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, buf := createRecoveryTestLogger()

	router := gin.New()
	router.Use(Recovery(logger))

	router.GET("/panic", func(c *gin.Context) {
		panic("test panic message")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Check panic was logged
	if buf.Len() == 0 {
		t.Error("Expected panic to be logged")
	}

	logOutput := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("panic")) {
		t.Errorf("Expected log to contain 'panic', got: %s", logOutput)
	}
}

func TestRecovery_NoPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := createRecoveryTestLogger()

	router := gin.New()
	router.Use(Recovery(logger))

	router.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/ok", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return 200
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", w.Body.String())
	}
}

func TestRecovery_PanicWithError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := createRecoveryTestLogger()

	router := gin.New()
	router.Use(Recovery(logger))

	router.GET("/panic", func(c *gin.Context) {
		var err error = &customError{message: "custom error"}
		panic(err)
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

type customError struct {
	message string
}

func (e *customError) Error() string {
	return e.message
}

func TestRecovery_MultipleRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := createRecoveryTestLogger()

	router := gin.New()
	router.Use(Recovery(logger))

	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	router.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// First request panics
	req1 := httptest.NewRequest("GET", "/panic", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusInternalServerError {
		t.Errorf("First request: expected status 500, got %d", w1.Code)
	}

	// Second request should still work
	req2 := httptest.NewRequest("GET", "/ok", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Second request: expected status 200, got %d", w2.Code)
	}
}

func TestRecovery_PanicWithNil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := createRecoveryTestLogger()

	router := gin.New()
	router.Use(Recovery(logger))

	router.GET("/panic", func(c *gin.Context) {
		panic(nil)
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()

	// Should not crash
	router.ServeHTTP(w, req)

	// Should return 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestRecovery_PanicWithInt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, _ := createRecoveryTestLogger()

	router := gin.New()
	router.Use(Recovery(logger))

	router.GET("/panic", func(c *gin.Context) {
		panic(42)
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

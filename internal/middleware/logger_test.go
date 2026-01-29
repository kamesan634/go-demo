package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func createTestLogger() (*zap.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	encoderConfig := zap.NewProductionEncoderConfig()
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	core := zapcore.NewCore(encoder, zapcore.AddSync(buf), zapcore.DebugLevel)
	logger := zap.New(core)
	return logger, buf
}

func TestLogger_LogsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, buf := createTestLogger()

	router := gin.New()
	router.Use(Logger(logger))

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Check that something was logged
	if buf.Len() == 0 {
		t.Error("Expected log output")
	}

	logOutput := buf.String()

	// Check log contains expected fields
	if !bytes.Contains(buf.Bytes(), []byte("GET")) {
		t.Errorf("Expected log to contain method, got: %s", logOutput)
	}

	if !bytes.Contains(buf.Bytes(), []byte("/test")) {
		t.Errorf("Expected log to contain path, got: %s", logOutput)
	}
}

func TestLogger_LogsStatusCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, buf := createTestLogger()

	router := gin.New()
	router.Use(Logger(logger))

	router.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	router.GET("/notfound", func(c *gin.Context) {
		c.String(http.StatusNotFound, "Not Found")
	})

	// Test 200
	req := httptest.NewRequest("GET", "/ok", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !bytes.Contains(buf.Bytes(), []byte("200")) {
		t.Errorf("Expected log to contain status 200")
	}

	buf.Reset()

	// Test 404
	req = httptest.NewRequest("GET", "/notfound", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !bytes.Contains(buf.Bytes(), []byte("404")) {
		t.Errorf("Expected log to contain status 404")
	}
}

func TestLogger_LogsLatency(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, buf := createTestLogger()

	router := gin.New()
	router.Use(Logger(logger))

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Check log contains latency field
	if !bytes.Contains(buf.Bytes(), []byte("latency")) {
		t.Error("Expected log to contain latency field")
	}
}

func TestLogger_LogsClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, buf := createTestLogger()

	router := gin.New()
	router.Use(Logger(logger))

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Check log contains IP field (the logger uses "ip" not "client_ip")
	if !bytes.Contains(buf.Bytes(), []byte(`"ip"`)) {
		t.Error("Expected log to contain ip field")
	}
}

func TestLogger_LogsUserAgent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, buf := createTestLogger()

	router := gin.New()
	router.Use(Logger(logger))

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Test-Agent/1.0")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Check log contains user agent
	if !bytes.Contains(buf.Bytes(), []byte("user_agent")) {
		t.Error("Expected log to contain user_agent field")
	}
}

func TestLogger_MultipleRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, buf := createTestLogger()

	router := gin.New()
	router.Use(Logger(logger))

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// Make multiple requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// Check that multiple logs were written
	logLines := bytes.Count(buf.Bytes(), []byte("\n"))
	if logLines < 5 {
		t.Errorf("Expected at least 5 log lines, got %d", logLines)
	}
}

func TestLogger_DifferentMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger, buf := createTestLogger()

	router := gin.New()
	router.Use(Logger(logger))

	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.POST("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.PUT("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.DELETE("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	methods := []string{"GET", "POST", "PUT", "DELETE"}

	for _, method := range methods {
		buf.Reset()

		req := httptest.NewRequest(method, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if !bytes.Contains(buf.Bytes(), []byte(method)) {
			t.Errorf("Expected log to contain method %s", method)
		}
	}
}

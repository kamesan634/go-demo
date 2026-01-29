package handler

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-demo/chat/internal/middleware"
	"github.com/go-demo/chat/internal/pkg/utils"
)

func setupUploadHandlerTest(t *testing.T) (*gin.Engine, *UploadHandler, *utils.JWTManager) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	handler := NewUploadHandler("http://localhost:8080")
	jwtManager := utils.NewJWTManager("test-secret", 15*time.Minute, 7*24*time.Hour, "test")

	router := gin.New()
	upload := router.Group("/api/v1/upload")
	upload.Use(middleware.Auth(jwtManager))
	{
		upload.POST("/image", handler.UploadImage)
		upload.POST("/file", handler.UploadFile)
		upload.POST("/avatar", handler.UploadAvatar)
	}

	return router, handler, jwtManager
}

func cleanupUploadTest(t *testing.T) {
	t.Helper()
	// Clean up test upload directories
	os.RemoveAll("./uploads")
}

func createMultipartRequest(t *testing.T, fieldName, filename string, content []byte, contentType string) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="` + fieldName + `"; filename="` + filename + `"`}
	h["Content-Type"] = []string{contentType}

	part, err := writer.CreatePart(h)
	if err != nil {
		t.Fatalf("Failed to create form part: %v", err)
	}

	if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
		t.Fatalf("Failed to write file content: %v", err)
	}

	writer.Close()
	return body, writer.FormDataContentType()
}

func TestUploadHandler_UploadImage(t *testing.T) {
	router, _, jwtManager := setupUploadHandlerTest(t)
	defer cleanupUploadTest(t)

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "alice")

	// Create a fake image content
	imageContent := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG magic bytes + padding
	for i := 0; i < 1000; i++ {
		imageContent = append(imageContent, byte(i%256))
	}

	body, contentType := createMultipartRequest(t, "file", "test.jpg", imageContent, "image/jpeg")

	req := httptest.NewRequest("POST", "/api/v1/upload/image", body)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUploadHandler_UploadImage_PNG(t *testing.T) {
	router, _, jwtManager := setupUploadHandlerTest(t)
	defer cleanupUploadTest(t)

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "alice")

	// PNG magic bytes
	imageContent := []byte{0x89, 0x50, 0x4E, 0x47}
	for i := 0; i < 500; i++ {
		imageContent = append(imageContent, byte(i%256))
	}

	body, contentType := createMultipartRequest(t, "file", "test.png", imageContent, "image/png")

	req := httptest.NewRequest("POST", "/api/v1/upload/image", body)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUploadHandler_UploadImage_InvalidType(t *testing.T) {
	router, _, jwtManager := setupUploadHandlerTest(t)
	defer cleanupUploadTest(t)

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "alice")

	// Try to upload a text file as image
	content := []byte("This is not an image")

	body, contentType := createMultipartRequest(t, "file", "test.txt", content, "text/plain")

	req := httptest.NewRequest("POST", "/api/v1/upload/image", body)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid image type, got %d", w.Code)
	}
}

func TestUploadHandler_UploadImage_TooLarge(t *testing.T) {
	router, _, jwtManager := setupUploadHandlerTest(t)
	defer cleanupUploadTest(t)

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "alice")

	// Create a file larger than 5MB
	largeContent := make([]byte, 6*1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	body, contentType := createMultipartRequest(t, "file", "large.jpg", largeContent, "image/jpeg")

	req := httptest.NewRequest("POST", "/api/v1/upload/image", body)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status 413 for large file, got %d", w.Code)
	}
}

func TestUploadHandler_UploadImage_NoFile(t *testing.T) {
	router, _, jwtManager := setupUploadHandlerTest(t)
	defer cleanupUploadTest(t)

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "alice")

	req := httptest.NewRequest("POST", "/api/v1/upload/image", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for no file, got %d", w.Code)
	}
}

func TestUploadHandler_UploadFile(t *testing.T) {
	router, _, jwtManager := setupUploadHandlerTest(t)
	defer cleanupUploadTest(t)

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "alice")

	// PDF magic bytes
	pdfContent := []byte("%PDF-1.4 test content")

	body, contentType := createMultipartRequest(t, "file", "document.pdf", pdfContent, "application/pdf")

	req := httptest.NewRequest("POST", "/api/v1/upload/file", body)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUploadHandler_UploadFile_TextFile(t *testing.T) {
	router, _, jwtManager := setupUploadHandlerTest(t)
	defer cleanupUploadTest(t)

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "alice")

	content := []byte("This is a plain text file for testing purposes.")

	body, contentType := createMultipartRequest(t, "file", "readme.txt", content, "text/plain")

	req := httptest.NewRequest("POST", "/api/v1/upload/file", body)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUploadHandler_UploadFile_TooLarge(t *testing.T) {
	router, _, jwtManager := setupUploadHandlerTest(t)
	defer cleanupUploadTest(t)

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "alice")

	// Create a file larger than 10MB
	largeContent := make([]byte, 11*1024*1024)

	body, contentType := createMultipartRequest(t, "file", "large.pdf", largeContent, "application/pdf")

	req := httptest.NewRequest("POST", "/api/v1/upload/file", body)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status 413 for large file, got %d", w.Code)
	}
}

func TestUploadHandler_UploadFile_InvalidType(t *testing.T) {
	router, _, jwtManager := setupUploadHandlerTest(t)
	defer cleanupUploadTest(t)

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "alice")

	content := []byte("#!/bin/bash\necho 'test'")

	body, contentType := createMultipartRequest(t, "file", "script.sh", content, "application/x-sh")

	req := httptest.NewRequest("POST", "/api/v1/upload/file", body)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid file type, got %d", w.Code)
	}
}

func TestUploadHandler_UploadAvatar(t *testing.T) {
	router, _, jwtManager := setupUploadHandlerTest(t)
	defer cleanupUploadTest(t)

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "alice")

	// Create a small image
	imageContent := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	for i := 0; i < 500; i++ {
		imageContent = append(imageContent, byte(i%256))
	}

	body, contentType := createMultipartRequest(t, "file", "avatar.jpg", imageContent, "image/jpeg")

	req := httptest.NewRequest("POST", "/api/v1/upload/avatar", body)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUploadHandler_UploadAvatar_TooLarge(t *testing.T) {
	router, _, jwtManager := setupUploadHandlerTest(t)
	defer cleanupUploadTest(t)

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "alice")

	// Create a file larger than 2MB
	largeContent := make([]byte, 3*1024*1024)

	body, contentType := createMultipartRequest(t, "file", "avatar.jpg", largeContent, "image/jpeg")

	req := httptest.NewRequest("POST", "/api/v1/upload/avatar", body)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status 413 for large avatar, got %d", w.Code)
	}
}

func TestUploadHandler_UploadAvatar_InvalidType(t *testing.T) {
	router, _, jwtManager := setupUploadHandlerTest(t)
	defer cleanupUploadTest(t)

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "alice")

	content := []byte("not an image")

	body, contentType := createMultipartRequest(t, "file", "avatar.txt", content, "text/plain")

	req := httptest.NewRequest("POST", "/api/v1/upload/avatar", body)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid avatar type, got %d", w.Code)
	}
}

func TestUploadHandler_Unauthorized(t *testing.T) {
	router, _, _ := setupUploadHandlerTest(t)
	defer cleanupUploadTest(t)

	req := httptest.NewRequest("POST", "/api/v1/upload/image", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestUploadHandler_DirectoryCreation(t *testing.T) {
	// Clean up first
	os.RemoveAll("./uploads")

	// Create handler - this should create directories
	handler := NewUploadHandler("http://localhost:8080")

	// Verify directories exist
	dirs := []string{
		filepath.Join(UploadDir, ImageSubDir),
		filepath.Join(UploadDir, FileSubDir),
		filepath.Join(UploadDir, AvatarSubDir),
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Expected directory %s to exist", dir)
		}
	}

	_ = handler // Use handler to avoid unused variable error
	cleanupUploadTest(t)
}

func TestUploadHandler_GIFImage(t *testing.T) {
	router, _, jwtManager := setupUploadHandlerTest(t)
	defer cleanupUploadTest(t)

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "alice")

	// GIF magic bytes
	gifContent := []byte("GIF89a")
	for i := 0; i < 500; i++ {
		gifContent = append(gifContent, byte(i%256))
	}

	body, contentType := createMultipartRequest(t, "file", "test.gif", gifContent, "image/gif")

	req := httptest.NewRequest("POST", "/api/v1/upload/image", body)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for GIF, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUploadHandler_WebPImage(t *testing.T) {
	router, _, jwtManager := setupUploadHandlerTest(t)
	defer cleanupUploadTest(t)

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "alice")

	// WebP file content
	webpContent := []byte("RIFF....WEBP")
	for i := 0; i < 500; i++ {
		webpContent = append(webpContent, byte(i%256))
	}

	body, contentType := createMultipartRequest(t, "file", "test.webp", webpContent, "image/webp")

	req := httptest.NewRequest("POST", "/api/v1/upload/image", body)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for WebP, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUploadHandler_UploadFile_Excel(t *testing.T) {
	router, _, jwtManager := setupUploadHandlerTest(t)
	defer cleanupUploadTest(t)

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "alice")

	content := []byte("Excel file content simulation")

	body, contentType := createMultipartRequest(t, "file", "data.xlsx", content, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	req := httptest.NewRequest("POST", "/api/v1/upload/file", body)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for Excel, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUploadHandler_UploadFile_Word(t *testing.T) {
	router, _, jwtManager := setupUploadHandlerTest(t)
	defer cleanupUploadTest(t)

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "alice")

	content := []byte("Word document content simulation")

	body, contentType := createMultipartRequest(t, "file", "document.docx", content, "application/vnd.openxmlformats-officedocument.wordprocessingml.document")

	req := httptest.NewRequest("POST", "/api/v1/upload/file", body)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for Word, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUploadHandler_UploadFile_Zip(t *testing.T) {
	router, _, jwtManager := setupUploadHandlerTest(t)
	defer cleanupUploadTest(t)

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "alice")

	// ZIP magic bytes
	content := []byte("PK\x03\x04 zip content simulation")

	body, contentType := createMultipartRequest(t, "file", "archive.zip", content, "application/zip")

	req := httptest.NewRequest("POST", "/api/v1/upload/file", body)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for ZIP, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUploadHandler_UploadFile_ImageAsFile(t *testing.T) {
	router, _, jwtManager := setupUploadHandlerTest(t)
	defer cleanupUploadTest(t)

	tokenPair, _ := jwtManager.GenerateTokenPair("user-123", "alice")

	// Images should also be allowed in the file upload endpoint
	imageContent := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	for i := 0; i < 500; i++ {
		imageContent = append(imageContent, byte(i%256))
	}

	body, contentType := createMultipartRequest(t, "file", "photo.jpg", imageContent, "image/jpeg")

	req := httptest.NewRequest("POST", "/api/v1/upload/file", body)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for image in file endpoint, got %d: %s", w.Code, w.Body.String())
	}
}

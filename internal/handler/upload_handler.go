package handler

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-demo/chat/internal/dto/response"
	"github.com/go-demo/chat/internal/middleware"
	"github.com/google/uuid"
)

const (
	MaxFileSize     = 10 << 20 // 10 MB
	MaxImageSize    = 5 << 20  // 5 MB
	UploadDir       = "./uploads"
	ImageSubDir     = "images"
	FileSubDir      = "files"
	AvatarSubDir    = "avatars"
)

var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

var allowedFileTypes = map[string]bool{
	"application/pdf":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	"text/plain":      true,
	"application/zip": true,
}

type UploadHandler struct {
	baseURL string
}

func NewUploadHandler(baseURL string) *UploadHandler {
	// Ensure upload directories exist
	dirs := []string{
		filepath.Join(UploadDir, ImageSubDir),
		filepath.Join(UploadDir, FileSubDir),
		filepath.Join(UploadDir, AvatarSubDir),
	}
	for _, dir := range dirs {
		os.MkdirAll(dir, 0755)
	}

	return &UploadHandler{
		baseURL: baseURL,
	}
}

// UploadImage godoc
// @Summary 上傳圖片
// @Description 上傳圖片檔案
// @Tags 上傳
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "圖片檔案"
// @Success 200 {object} response.Response{data=map[string]string}
// @Failure 400 {object} response.Response
// @Failure 413 {object} response.Response
// @Router /api/v1/upload/image [post]
func (h *UploadHandler) UploadImage(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "無法讀取檔案")
		return
	}
	defer file.Close()

	// Check file size
	if header.Size > MaxImageSize {
		response.ErrorWithStatus(c, 413, "圖片大小不能超過 5MB")
		return
	}

	// Check content type
	contentType := header.Header.Get("Content-Type")
	if !allowedImageTypes[contentType] {
		response.BadRequest(c, "不支援的圖片格式，請上傳 JPEG、PNG、GIF 或 WebP 格式")
		return
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s_%d%s", uuid.New().String(), time.Now().Unix(), ext)
	filePath := filepath.Join(UploadDir, ImageSubDir, filename)

	// Save file
	if err := h.saveFile(file, filePath); err != nil {
		response.InternalError(c, "儲存檔案失敗")
		return
	}

	fileURL := fmt.Sprintf("%s/uploads/%s/%s", h.baseURL, ImageSubDir, filename)

	response.Success(c, gin.H{
		"url":      fileURL,
		"filename": header.Filename,
		"size":     header.Size,
		"type":     contentType,
	})
}

// UploadFile godoc
// @Summary 上傳檔案
// @Description 上傳一般檔案
// @Tags 上傳
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "檔案"
// @Success 200 {object} response.Response{data=map[string]string}
// @Failure 400 {object} response.Response
// @Failure 413 {object} response.Response
// @Router /api/v1/upload/file [post]
func (h *UploadHandler) UploadFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "無法讀取檔案")
		return
	}
	defer file.Close()

	// Check file size
	if header.Size > MaxFileSize {
		response.ErrorWithStatus(c, 413, "檔案大小不能超過 10MB")
		return
	}

	// Check content type
	contentType := header.Header.Get("Content-Type")
	if !allowedFileTypes[contentType] && !allowedImageTypes[contentType] {
		response.BadRequest(c, "不支援的檔案格式")
		return
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	safeName := strings.ReplaceAll(header.Filename, " ", "_")
	filename := fmt.Sprintf("%s_%s", uuid.New().String()[:8], safeName)
	if len(filename) > 100 {
		filename = fmt.Sprintf("%s%s", uuid.New().String(), ext)
	}
	filePath := filepath.Join(UploadDir, FileSubDir, filename)

	// Save file
	if err := h.saveFile(file, filePath); err != nil {
		response.InternalError(c, "儲存檔案失敗")
		return
	}

	fileURL := fmt.Sprintf("%s/uploads/%s/%s", h.baseURL, FileSubDir, filename)

	response.Success(c, gin.H{
		"url":      fileURL,
		"filename": header.Filename,
		"size":     header.Size,
		"type":     contentType,
	})
}

// UploadAvatar godoc
// @Summary 上傳頭像
// @Description 上傳用戶頭像
// @Tags 上傳
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "頭像圖片"
// @Success 200 {object} response.Response{data=map[string]string}
// @Failure 400 {object} response.Response
// @Failure 413 {object} response.Response
// @Router /api/v1/upload/avatar [post]
func (h *UploadHandler) UploadAvatar(c *gin.Context) {
	userID := middleware.GetUserID(c)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "無法讀取檔案")
		return
	}
	defer file.Close()

	// Check file size (2MB for avatars)
	if header.Size > 2<<20 {
		response.ErrorWithStatus(c, 413, "頭像大小不能超過 2MB")
		return
	}

	// Check content type
	contentType := header.Header.Get("Content-Type")
	if !allowedImageTypes[contentType] {
		response.BadRequest(c, "不支援的圖片格式，請上傳 JPEG、PNG、GIF 或 WebP 格式")
		return
	}

	// Generate filename using user ID
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s_%d%s", userID, time.Now().Unix(), ext)
	filePath := filepath.Join(UploadDir, AvatarSubDir, filename)

	// Save file
	if err := h.saveFile(file, filePath); err != nil {
		response.InternalError(c, "儲存檔案失敗")
		return
	}

	fileURL := fmt.Sprintf("%s/uploads/%s/%s", h.baseURL, AvatarSubDir, filename)

	response.Success(c, gin.H{
		"url":      fileURL,
		"filename": header.Filename,
		"size":     header.Size,
		"type":     contentType,
	})
}

func (h *UploadHandler) saveFile(file io.Reader, path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	return err
}

package handler

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	uploader       *GCSUploader
	requestTimeout = 30 * time.Second
)

type ApiResponse struct {
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func ConnectGCS(credentialPath, bucketName string) error {
	creds, err := os.ReadFile(credentialPath)
	if err != nil {
		return err
	}

	uploader = NewGCSUploader(creds, bucketName)
	if err := uploader.Init(); err != nil {
		return err
	}
	return nil
}

func UploadFile(c *gin.Context) {
	filename := strings.TrimSpace(c.Query("filename"))
	folder := strings.TrimSpace(c.Query("folder"))

	if filename == "" || folder == "" {
		c.JSON(http.StatusBadRequest, ApiResponse{Error: "filename and folder are required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	objectname := path.Join(folder, filepath.Base(filename))

	uploadSize, err := uploader.UploadFile(ctx, filename, objectname, 0, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{Error: err.Error()})
		return
	}

	size := fmt.Sprintf("%d bytes", uploadSize)
	c.JSON(http.StatusCreated, ApiResponse{Message: "File uploaded successfully", Data: map[string]string{"path": objectname, "size": size}})
}

func DownloadFile(c *gin.Context) {
	objectname := strings.TrimSpace(c.Query("objectname"))
	destination := strings.TrimSpace(c.Query("destination"))

	if objectname == "" {
		c.JSON(http.StatusBadRequest, ApiResponse{Error: "objectname is required"})
		return
	}

	if destination == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			c.JSON(http.StatusInternalServerError, ApiResponse{Error: "Failed to get user home directory"})
			return
		}
		destination = filepath.Join(home, "Downloads")
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	downloadSize, err := uploader.DownloadFile(ctx, objectname, destination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{Error: err.Error()})
		return
	}

	size := fmt.Sprintf("%d bytes", downloadSize)
	c.JSON(http.StatusOK, ApiResponse{Message: "File downloaded successfully", Data: map[string]string{"path": destination, "size": size}})
}

func ListFiles(c *gin.Context) {
	folder := strings.TrimSpace(c.Query("folder"))

	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	files, err := uploader.ListObjects(ctx, folder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{Error: err.Error()})
		return
	}

	if len(files) > 0 {
		c.JSON(http.StatusOK, ApiResponse{Message: "Files found", Data: files})
		return
	}

	c.JSON(http.StatusOK, ApiResponse{Message: "No files found"})
}

func DeleteObject(c *gin.Context) {
	objectname := strings.TrimSpace(c.Query("objectname"))

	if objectname == "" {
		c.JSON(http.StatusBadRequest, ApiResponse{Error: "objectname is required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	err := uploader.DeleteObject(ctx, objectname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ApiResponse{Message: "File deleted successfully", Data: map[string]string{"path": objectname}})
}

func UploadBuffer(c *gin.Context) {
	objectname := strings.TrimSpace(c.Query("objectname"))

	if objectname == "" {
		c.JSON(http.StatusBadRequest, ApiResponse{Error: "objectname is required"})
		return
	}

	data, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{Error: err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	uploadSize, err := uploader.UploadBuffer(ctx, data, objectname, 0, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{Error: err.Error()})
		return
	}
	size := fmt.Sprintf("%d bytes", uploadSize)
	c.JSON(http.StatusCreated, ApiResponse{Message: "Buffer uploaded successfully", Data: map[string]string{"path": objectname, "size": size}})
}

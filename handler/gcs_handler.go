package handler

import (
	"context"
	"net/http"
	"os"
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
	filename := c.Query("filename")
	objectname := c.Query("objectname")

	if filename == "" || objectname == "" {
		c.JSON(http.StatusBadRequest, ApiResponse{Error: "filename and objectname are required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	_, err := uploader.UploadFile(ctx, filename, objectname, 0, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ApiResponse{Message: "File uploaded successfully", Data: map[string]string{"path": objectname}})
}

func DownloadFile(c *gin.Context) {
	objectname := c.Query("objectname")
	destination := c.Query("destination")

	if objectname == "" || destination == "" {
		c.JSON(http.StatusBadRequest, ApiResponse{Error: "objectname and destination are required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	_, err := uploader.DownloadFile(ctx, objectname, destination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ApiResponse{Message: "File downloaded successfully", Data: map[string]string{"path": destination}})
}

func ListFiles(c *gin.Context) {
	prefix := c.Query("prefix")

	ctx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
	defer cancel()

	files, err := uploader.ListObjects(ctx, prefix)
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
	objectname := c.Query("objectname")

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
	objectname := c.Query("objectname")

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

	_, err = uploader.UploadBuffer(ctx, data, objectname, 0, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ApiResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ApiResponse{Message: "Buffer uploaded successfully", Data: map[string]string{"path": objectname}})
}

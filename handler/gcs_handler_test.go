package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gcsuploader/handler"

	"github.com/gin-gonic/gin"
)

type apiResp struct {
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/upload-buffer", handler.UploadBuffer)
	r.POST("/upload-file", handler.UploadFile)
	r.GET("/list-files", handler.ListFiles)
	r.GET("/download-file", handler.DownloadFile)
	r.DELETE("/delete-object", handler.DeleteObject)

	if err := handler.ConnectGCS("/home/akash/Downloads/gcsuploader/config/credentials.json", "dmtfota"); err != nil {
		t.Fatalf("ConnectGCS failed: %v", err)
	}
	return httptest.NewServer(r)
}

func parseResp(t *testing.T, res *http.Response) apiResp {
	t.Helper()
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	var ar apiResp
	if err := json.Unmarshal(body, &ar); err != nil {
		t.Fatalf("unmarshal failed: %v (body: %s)", err, string(body))
	}
	return ar
}

func getDataMap(t *testing.T, ar apiResp) map[string]interface{} {
	t.Helper()
	m, ok := ar.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be an object, got %#v", ar.Data)
	}
	return m
}

func TestAPIIntegration(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	client := &http.Client{Timeout: 30 * time.Second}

	folder := fmt.Sprintf("api-test/%d", time.Now().UnixNano())
	object1 := folder + "/buf.txt"
	object2 := folder + "/file.txt"

	t.Run("UploadBuffer", func(t *testing.T) {
		req, _ := http.NewRequest("POST", srv.URL+"/upload-buffer?objectname="+object1, bytes.NewBufferString("hello buffer"))
		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d", res.StatusCode)
		}
		ar := parseResp(t, res)
		if ar.Error != "" {
			t.Fatalf("unexpected error: %s", ar.Error)
		}
		if ar.Message == "" || !strings.Contains(ar.Message, "Buffer uploaded successfully") {
			t.Fatalf("unexpected message: %q", ar.Message)
		}
		data := getDataMap(t, ar)
		if data["path"] != object1 {
			t.Fatalf("expected path %q, got %v", object1, data["path"])
		}
		if _, ok := data["size"].(string); !ok {
			t.Fatalf("expected size string, got %#v", data["size"])
		}
	})

	t.Run("UploadFile", func(t *testing.T) {
		tmpFile := filepath.Join(os.TempDir(), "api_upload.txt")
		if err := os.WriteFile(tmpFile, []byte("file upload test"), 0644); err != nil {
			t.Fatalf("write temp file failed: %v", err)
		}

		expectedPath := path.Join(folder, filepath.Base(tmpFile))
		object2 = expectedPath

		req, _ := http.NewRequest("POST",
			srv.URL+"/upload-file?filename="+tmpFile+"&folder="+folder, nil)
		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d", res.StatusCode)
		}
		ar := parseResp(t, res)
		if ar.Error != "" {
			t.Fatalf("unexpected error: %s", ar.Error)
		}
		if ar.Message == "" || !strings.Contains(ar.Message, "File uploaded successfully") {
			t.Fatalf("unexpected message: %q", ar.Message)
		}
		data := getDataMap(t, ar)
		if data["path"] != expectedPath {
			t.Fatalf("expected path %q, got %v", expectedPath, data["path"])
		}
		if size, ok := data["size"].(string); !ok || !strings.HasSuffix(size, "bytes") {
			t.Fatalf("expected size string ending with 'bytes', got %#v", data["size"])
		}
	})

	t.Run("ListFiles_FilesFound", func(t *testing.T) {
		req, _ := http.NewRequest("GET", srv.URL+"/list-files?folder="+folder, nil)
		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}
		ar := parseResp(t, res)
		if ar.Error != "" {
			t.Fatalf("unexpected error: %s", ar.Error)
		}
		if ar.Message == "" || !strings.Contains(ar.Message, "Files found") {
			t.Fatalf("unexpected message: %q", ar.Message)
		}
		files, ok := ar.Data.([]interface{})
		if !ok || len(files) == 0 {
			t.Fatalf("expected non-empty files list, got %#v", ar.Data)
		}
	})

	t.Run("DownloadFile_WithDestination", func(t *testing.T) {
		destDir := t.TempDir()
		req, _ := http.NewRequest("GET", srv.URL+"/download-file?objectname="+object1+"&destination="+destDir, nil)
		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}
		ar := parseResp(t, res)
		if ar.Error != "" {
			t.Fatalf("unexpected error: %s", ar.Error)
		}
		if ar.Message == "" || !strings.Contains(ar.Message, "File downloaded successfully") {
			t.Fatalf("unexpected message: %q", ar.Message)
		}
		data := getDataMap(t, ar)
		if data["path"] != destDir {
			t.Fatalf("expected path %q, got %v", destDir, data["path"])
		}
		if _, ok := data["size"].(string); !ok {
			t.Fatalf("expected size string, got %#v", data["size"])
		}
		expectedFile := filepath.Join(destDir, filepath.Base(object1))
		if _, err := os.Stat(expectedFile); err != nil {
			t.Fatalf("downloaded file missing: %v", err)
		}
		dataBytes, _ := os.ReadFile(expectedFile)
		if !strings.Contains(string(dataBytes), "hello buffer") {
			t.Fatalf("unexpected file content: %s", string(dataBytes))
		}
	})

	t.Run("ListFiles_NoFilesFound", func(t *testing.T) {
		emptyFolder := fmt.Sprintf("api-test-empty/%d", time.Now().UnixNano())
		req, _ := http.NewRequest("GET", srv.URL+"/list-files?folder="+emptyFolder, nil)
		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}
		ar := parseResp(t, res)
		if ar.Error != "" {
			t.Fatalf("unexpected error: %s", ar.Error)
		}
		if ar.Message == "" || !strings.Contains(ar.Message, "No files found") {
			t.Fatalf("unexpected message: %q", ar.Message)
		}
	})

	t.Run("DeleteObject1", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", srv.URL+"/delete-object?objectname="+object1, nil)
		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}
		ar := parseResp(t, res)
		if ar.Error != "" {
			t.Fatalf("unexpected error: %s", ar.Error)
		}
		if ar.Message == "" || !strings.Contains(ar.Message, "File deleted successfully") {
			t.Fatalf("unexpected message: %q", ar.Message)
		}
		data := getDataMap(t, ar)
		if data["path"] != object1 {
			t.Fatalf("expected path %q, got %v", object1, data["path"])
		}
	})

	t.Run("DeleteObject2", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", srv.URL+"/delete-object?objectname="+object2, nil)
		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}
		ar := parseResp(t, res)
		if ar.Error != "" {
			t.Fatalf("unexpected error: %s", ar.Error)
		}
		if ar.Message == "" || !strings.Contains(ar.Message, "File deleted successfully") {
			t.Fatalf("unexpected message: %q", ar.Message)
		}
		data := getDataMap(t, ar)
		if data["path"] != object2 {
			t.Fatalf("expected path %q, got %v", object2, data["path"])
		}
	})
}

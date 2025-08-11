package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gcsuploader/handler"

	"github.com/gin-gonic/gin"
)

type apiResp struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error"`
}

func newTestServer(t *testing.T) *httptest.Server {
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
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	var ar apiResp
	if err := json.Unmarshal(body, &ar); err != nil {
		t.Fatalf("unmarshal failed: %v (body: %s)", err, string(body))
	}
	return ar
}

func TestAPIIntegration(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	prefix := fmt.Sprintf("api-test/%d/", time.Now().UnixNano())
	object1 := prefix + "buf.txt"
	object2 := prefix + "file.txt"

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
	})

	t.Run("UploadFile", func(t *testing.T) {
		tmpFile := filepath.Join(os.TempDir(), "api_upload.txt")
		if err := os.WriteFile(tmpFile, []byte("file upload test"), 0644); err != nil {
			t.Fatalf("write temp file failed: %v", err)
		}
		req, _ := http.NewRequest("POST", srv.URL+"/upload-file?filename="+tmpFile+"&objectname="+object2, nil)
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
	})

	t.Run("ListFiles", func(t *testing.T) {
		req, _ := http.NewRequest("GET", srv.URL+"/list-files?prefix="+prefix, nil)
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
		files, ok := ar.Data.([]interface{})
		if !ok || len(files) == 0 {
			t.Fatalf("expected files list, got %#v", ar.Data)
		}
	})

	t.Run("DownloadFile", func(t *testing.T) {
		dest := filepath.Join(os.TempDir(), "api_download.txt")
		req, _ := http.NewRequest("GET", srv.URL+"/download-file?objectname="+object1+"&destination="+dest, nil)
		res, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}
		if _, err := os.Stat(dest); err != nil {
			t.Fatalf("downloaded file missing: %v", err)
		}
		data, _ := os.ReadFile(dest)
		if !strings.Contains(string(data), "hello buffer") {
			t.Fatalf("unexpected file content: %s", string(data))
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
	})
}

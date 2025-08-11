package handler_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gcsuploader/handler"
)

var (
	testBucket      = "dmtfota"
	testCredentials = "/home/akash/Downloads/gcsuploader/config/credentials.json"
)

func requireEnv(t *testing.T) {
	if testBucket == "" || testCredentials == "" {
		t.Skip("GCS_TEST_BUCKET and GCS_CREDENTIALS_FILE must be set for integration tests")
	}
}

func newUploader(t *testing.T) *handler.GCSUploader {
	creds, err := os.ReadFile(testCredentials)
	if err != nil {
		t.Fatalf("failed to read credentials: %v", err)
	}

	uploader := handler.NewGCSUploader(creds, testBucket)
	if err := uploader.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	return uploader
}

func TestGCSUploaderIntegration(t *testing.T) {
	requireEnv(t)
	ctx := context.Background()

	uploader := newUploader(t)

	prefix := fmt.Sprintf("test-integration/%d/", time.Now().UnixNano())
	objectName := prefix + "hello.txt"
	downloadedFile := filepath.Join(os.TempDir(), "gcs_test_download.txt")

	t.Run("UploadBuffer", func(t *testing.T) {
		content := []byte("Hello, GCS!")
		n, err := uploader.UploadBuffer(ctx, content, objectName, 0, func(n int64) {})
		if err != nil {
			t.Fatalf("UploadBuffer failed: %v", err)
		}
		if int(n) != len(content) {
			t.Fatalf("expected %d bytes copied, got %d", len(content), n)
		}
	})

	t.Run("ListObjects", func(t *testing.T) {
		objs, err := uploader.ListObjects(ctx, prefix)
		if err != nil {
			t.Fatalf("ListObjects failed: %v", err)
		}
		found := false
		for _, obj := range objs {
			if obj == objectName {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected object %q in list, got %v", objectName, objs)
		}
	})

	t.Run("DownloadFile", func(t *testing.T) {
		n, err := uploader.DownloadFile(ctx, objectName, downloadedFile)
		if err != nil {
			t.Fatalf("DownloadFile failed: %v", err)
		}
		if n == 0 {
			t.Fatal("expected downloaded bytes > 0")
		}
		data, err := os.ReadFile(downloadedFile)
		if err != nil {
			t.Fatalf("reading downloaded file failed: %v", err)
		}
		if !strings.Contains(string(data), "Hello, GCS!") {
			t.Fatalf("unexpected file content: %s", string(data))
		}
	})

	t.Run("UploadFile", func(t *testing.T) {
		tmpFile := filepath.Join(os.TempDir(), "gcs_test_upload.txt")
		content := []byte("File upload test")
		if err := os.WriteFile(tmpFile, content, 0644); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}

		objectName2 := prefix + "file_upload.txt"
		n, err := uploader.UploadFile(ctx, tmpFile, objectName2, 0, func(n int64) {})
		if err != nil {
			t.Fatalf("UploadFile failed: %v", err)
		}
		if int(n) != len(content) {
			t.Fatalf("expected %d bytes copied, got %d", len(content), n)
		}
	})

	t.Run("DeleteObject", func(t *testing.T) {
		if err := uploader.DeleteObject(ctx, objectName); err != nil {
			t.Fatalf("DeleteObject failed: %v", err)
		}

		// Ensure it’s gone
		objs, err := uploader.ListObjects(ctx, prefix)
		if err != nil {
			t.Fatalf("ListObjects failed: %v", err)
		}
		for _, obj := range objs {
			if obj == objectName {
				t.Fatalf("object %q still exists after delete", objectName)
			}
		}
	})
}

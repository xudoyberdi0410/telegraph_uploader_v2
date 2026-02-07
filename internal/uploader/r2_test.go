package uploader

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"telegraph_uploader_v2/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// NOTE: createTestImage is defined in image_processor_test.go

func TestNew(t *testing.T) {
	cfg := &config.Config{
		R2AccountId: "acc",
		R2AccessKey: "key",
		R2SecretKey: "secret",
	}
	// minio.New performs some validation on endpoint format
	u, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if u.client == nil {
		t.Error("client is nil")
	}
	if u.cfg != cfg {
		t.Error("config not stored")
	}
}

func TestUploadChapter(t *testing.T) {
	// Mock S3 server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read body to prevent client complaints
		_, _ = io.Copy(io.Discard, r.Body)

		if r.Method == "PUT" {
			w.Header().Set("ETag", "\"1234567890abcdef\"")
			w.WriteHeader(http.StatusOK)
			return
		}
		
		if r.Method == "HEAD" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Name>test_bucket</Name><Prefix></Prefix><Marker></Marker><MaxKeys>1000</MaxKeys><IsTruncated>false</IsTruncated><Contents></Contents></ListBucketResult>`))
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Setup Uploader with Mock
	cfg := &config.Config{
		BucketName:   "test_bucket",
		PublicDomain: "http://test.com",
	}

	endpoint := ts.Listener.Addr().String()
	minioClient, _ := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4("key", "secret", ""),
		Secure: false,
	})

	uploader := NewWithClient(minioClient, cfg, nil)

	// Create dummy image file
	tmpDir, err := os.MkdirTemp("", "uploadtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	imgPath := createTestImage(t, tmpDir, "upload.png", 100, 100)

	// Valid upload
	result := uploader.UploadChapter(context.Background(), []string{imgPath}, ResizeSettings{}, nil)
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if len(result.Links) != 1 {
		t.Error("expected 1 link")
	}
	if !strings.HasPrefix(result.Links[0], "http://test.com/") {
		t.Errorf("expected link to have public domain, got %s", result.Links[0])
	}

	// Invalid file upload
	result = uploader.UploadChapter(context.Background(), []string{"nonexistent.png"}, ResizeSettings{}, nil)
	if result.Success {
		t.Error("expected failure for nonexistent file")
	}
}

func TestUploadChapter_UploadError(t *testing.T) {
	// Server that fails
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	endpoint := ts.Listener.Addr().String()
	minioClient, _ := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4("key", "secret", ""),
		Secure: false,
	})
	
	uploader := NewWithClient(minioClient, &config.Config{BucketName: "b"}, nil)

	tmpDir, _ := os.MkdirTemp("", "uploadtest_fail")
	defer os.RemoveAll(tmpDir)
	imgPath := createTestImage(t, tmpDir, "fail.png", 10, 10)

	result := uploader.UploadChapter(context.Background(), []string{imgPath}, ResizeSettings{}, nil)
	if result.Success {
		t.Error("expected failure when server errors")
	}
	if !strings.Contains(result.Error, "Upload error") {
		t.Errorf("expected Upload error message, got %s", result.Error)
	}
}

func TestUploadChapter_PartialFailure(t *testing.T) {
	// One good file, one bad file
	tmpDir, _ := os.MkdirTemp("", "uploadtest_partial")
	defer os.RemoveAll(tmpDir)
	
	goodPath := createTestImage(t, tmpDir, "good.png", 10, 10)

	badPath := "nonexistent.png"

	// Mock server always succeeds for the good file
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	endpoint := ts.Listener.Addr().String()
	minioClient, _ := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4("key", "secret", ""),
		Secure: false,
	})
	
	uploader := NewWithClient(minioClient, &config.Config{BucketName: "b", PublicDomain: "http://d"}, nil)

	result := uploader.UploadChapter(context.Background(), []string{goodPath, badPath}, ResizeSettings{}, nil)
	
	// Expect failure because at least one failed
	if result.Success {
		t.Error("expected failure for partial error")
	}

	// Check error message contains info
	if !strings.Contains(result.Error, "Processing failed") && !strings.Contains(result.Error, "open error") && !strings.Contains(result.Error, "Hash error") && !strings.Contains(result.Error, "Read error") {
		t.Errorf("expected error details, got %s", result.Error)
	}
}
package uploader

import (
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

func TestNew(t *testing.T) {
	cfg := &config.Config{
		R2AccountId: "acc",
		R2AccessKey: "key",
		R2SecretKey: "secret",
	}
	// New uses real network potentially to discover endpoint? 
	// No, it just configures client. But minio.New constructs URL.
	// We can't easily mock the validation inside minio.New unless we pass bad args.
	
	u, err := New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if u.client == nil {
		t.Error("client is nil")
	}
}

func TestUploadChapter(t *testing.T) {
	// Mock S3 server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read body
		_, _ = io.Copy(io.Discard, r.Body)

		if r.Method == "PUT" {
			w.Header().Set("ETag", "\"1234567890abcdef\"")
			w.WriteHeader(http.StatusOK)
			return
		}
		
		// Handle potential BucketExists check (HEAD /bucket)
		if r.Method == "HEAD" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		// Handle potential MakeBucket (PUT /bucket) - should return 200 OK
		// But in typical flow we just PutObject.

		// For GET requests (bucket check/listing), return valid XML
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

	uploader := NewWithClient(minioClient, cfg)

	// Create dummy image file
	tmpDir, err := os.MkdirTemp("", "uploadtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	imgPath := createTestImage(t, tmpDir, "upload.png", 100, 100)

	// Valid upload
	result := uploader.UploadChapter([]string{imgPath}, ResizeSettings{})
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if len(result.Links) != 1 {
		t.Error("expected 1 link")
	}

	// Invalid file upload
	result = uploader.UploadChapter([]string{"nonexistent.png"}, ResizeSettings{})
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
	
	uploader := NewWithClient(minioClient, &config.Config{BucketName: "b"})

	tmpDir, _ := os.MkdirTemp("", "uploadtest_fail")
	defer os.RemoveAll(tmpDir)
	imgPath := createTestImage(t, tmpDir, "fail.png", 10, 10)

	result := uploader.UploadChapter([]string{imgPath}, ResizeSettings{})
	if result.Success {
		t.Error("expected failure when server errors")
	}
	if !strings.Contains(result.Error, "Upload error") {
		t.Errorf("expected Upload error message, got %s", result.Error)
	}
}

// Reuse helper
// Note: createTestImage is in image_processor_test.go which is in same package 'uploader', so it is visible if in same directory?
// Yes, `go test ./internal/uploader` compiles all *_test.go files together.
// So I don't need to redeclare it if I don't overwrite image_processor_test.go.
// BUT `image_processor_test.go` was created via write_to_file in previous step.
// I should make sure I don't delete it. I am overwriting `r2_test.go`.

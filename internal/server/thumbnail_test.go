package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestFileLoader_ServeHTTP(t *testing.T) {
	handler := NewFileLoader()

	// Create a temp file to serve
	tmpFile, err := os.CreateTemp("", "thumb*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("image content"))
	tmpFile.Close()

	// Construct request: /thumbnail/<path>
	// Path must be url encoded if it contains special chars, but for temp file on windows it might be tricky with backslashes.
	// The server implementation uses `url.QueryUnescape(strings.TrimPrefix(req.URL.Path, "/thumbnail/"))`
	// So we should strictly pass the raw path appended. But Windows paths have backslashes.
	// `filepath.ToSlash` might be useful if the server handled it, but server does `os.Open(decodedPath)`.
	// On Windows `os.Open` handles forward slashes too usually? Let's assume standard behavior.
	// Ideally we pass the exact path.
	
	reqPath := "/thumbnail/" + tmpFile.Name()
	
	req, err := http.NewRequest("GET", reqPath, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if rr.Body.String() != "image content" {
		t.Errorf("expected body 'image content', got %s", rr.Body.String())
	}
}

func TestFileLoader_NotFound(t *testing.T) {
	handler := NewFileLoader()
	req, _ := http.NewRequest("GET", "/thumbnail/nonexistent/path.jpg", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestFileLoader_InvalidPrefix(t *testing.T) {
	handler := NewFileLoader()
	req, _ := http.NewRequest("GET", "/other/path", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for invalid prefix, got %d", rr.Code)
	}
}

func TestFileLoader_BadURL(t *testing.T) {
	handler := NewFileLoader()
	// To test QueryUnescape error, we need a path that is accepted by string prefix check check 
	// but fails QueryUnescape.
	// HasPrefix checks req.URL.Path. QueryUnescape checks rawPath (trim prefix).
	// QueryUnescape fails on '%' followed by incomplete escape.
	// We can manually set the URL path on a request object without parsing from string.
	req, _ := http.NewRequest("GET", "/", nil)
	req.URL.Path = "/thumbnail/%"
	
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad url, got %d", rr.Code)
	}
}

type FailWriter struct {
	headers http.Header
}

func (w *FailWriter) Header() http.Header {
	if w.headers == nil {
		w.headers = make(http.Header)
	}
	return w.headers
}

func (w *FailWriter) Write(b []byte) (int, error) {
	return 0, errors.New("write error")
}

func (w *FailWriter) WriteHeader(statusCode int) {}

func TestFileLoader_WriteError(t *testing.T) {
	handler := NewFileLoader()
	
	// Create a temp file to serve
	tmpFile, err := os.CreateTemp("", "thumb*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write([]byte("image data"))
	tmpFile.Close()

	req, _ := http.NewRequest("GET", "/thumbnail/"+url.QueryEscape(tmpFile.Name()), nil)
	fw := &FailWriter{}

	handler.ServeHTTP(fw, req)
	// We just want to ensure it doesn't panic and executes the error path (which logs)
	// Coverage will show if it hit.
}

func TestFileLoader_Extensions(t *testing.T) {
	handler := NewFileLoader()

	tests := []struct {
		name         string
		path         string
		expectedCode int
	}{
		{"Allowed JPG", "test.jpg", http.StatusNotFound}, // 404 because file doesn't exist, but passed extension check
		{"Allowed PNG", "test.png", http.StatusNotFound},
		{"Allowed WEBP", "test.webp", http.StatusNotFound},
		{"Disallowed TXT", "test.txt", http.StatusForbidden},
		{"Disallowed No Ext", "testfile", http.StatusForbidden},
		{"Disallowed PHP", "script.php", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/thumbnail/"+tt.path, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != tt.expectedCode {
				t.Errorf("%s: expected status %d, got %d", tt.name, tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestFileLoader_IsDirectory(t *testing.T) {
	handler := NewFileLoader()

	// Use current directory
	req, _ := http.NewRequest("GET", "/thumbnail/.", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// "." will be Cleaned to "." which has no allowed extension, so it should be forbidden by extension check first
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for directory '.', got %d", rr.Code)
	}
}

func TestFileLoader_DirectoryWithImageExt(t *testing.T) {
	handler := NewFileLoader()

	dirName := "testdir.jpg"
	err := os.Mkdir(dirName, 0755)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dirName)

	req, _ := http.NewRequest("GET", "/thumbnail/"+dirName, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for directory, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Is a directory") {
		t.Errorf("expected 'Is a directory' error message, got %s", rr.Body.String())
	}
}

func TestFileLoader_PathTraversal(t *testing.T) {
	handler := NewFileLoader()

	tests := []struct {
		name string
		path string
	}{
		{"Simple traversal", "/thumbnail/../../etc/passwd"},
		{"Encoded traversal", "/thumbnail/..%2f..%2fetc%2fpasswd"},
		{"Traversal with image ext", "/thumbnail/../../etc/passwd.jpg"},
		{"Mixed slashes", "/thumbnail/..\\..\\etc\\passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.path, nil)
			// Manually set Path to avoid cleaning by NewRequest
			req.URL.Path = tt.path

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusForbidden {
				t.Errorf("%s: expected 403, got %d", tt.name, rr.Code)
			}
		})
	}
}

package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
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

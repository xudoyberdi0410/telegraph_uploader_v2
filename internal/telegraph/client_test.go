package telegraph

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreatePage(t *testing.T) {
	// Mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/createPage" {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"ok": true, "result": {"url": "http://telegra.ph/test-123"}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := &Client{
		Token:   "test_token",
		BaseURL: ts.URL,
	}

	url := client.CreatePage("Test Title", []string{"http://img1.jpg"})
	if url != "http://telegra.ph/test-123" {
		t.Errorf("expected url http://telegra.ph/test-123, got %s", url)
	}
}

func TestCreatePage_NoToken(t *testing.T) {
	// Mock server handling createAccount AND createPage
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/createAccount" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"ok": true, "result": {"access_token": "new_generated_token"}}`))
			return
		}
		if r.URL.Path == "/createPage" {
			// Verify we are using the new token
			if r.FormValue("access_token") != "new_generated_token" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"ok": true, "result": {"url": "http://telegra.ph/created-with-new-token"}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := &Client{
		Token:   "", // Empty token
		BaseURL: ts.URL,
	}

	url := client.CreatePage("Title", []string{})
	if url != "http://telegra.ph/created-with-new-token" {
		t.Errorf("expected url with new token, got %s", url)
	}
	// Verify token was stored
	if client.Token != "new_generated_token" {
		t.Errorf("expected client token to be updated, got %s", client.Token)
	}
}

func TestCreatePage_CreateAccountFail(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/createAccount" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"ok": false, "error": "FAIL"}`))
			return
		}
	}))
	defer ts.Close()

	client := &Client{
		Token:   "",
		BaseURL: ts.URL,
	}

	url := client.CreatePage("Title", nil)
	// Expect error string
	if !strings.HasPrefix(url, "Ошибка создания аккаунта") {
		t.Errorf("expected account creation error, got %s", url)
	}
}


func TestEditPage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/editPage" {
			t.Errorf("expected path /editPage, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok": true, "result": {"url": "http://telegra.ph/test-path"}}`))
	}))
	defer ts.Close()

	client := &Client{
		Token:   "test_token",
		BaseURL: ts.URL,
	}

	url := client.EditPage("test-path", "New Title", []string{}, "custom_token")
	if url != "http://telegra.ph/test-path" {
		t.Errorf("expected success url, got %s", url)
	}
}

func TestEditPage_UseConfigToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("access_token") != "default" {
			t.Errorf("expected default token, got %s", r.FormValue("access_token"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok": true, "result": {"url": "ok"}}`))
	}))
	defer ts.Close()

	client := &Client{Token: "default", BaseURL: ts.URL}
	client.EditPage("path", "title", nil, "") // Empty access token
}

func TestGetPage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := `
		{
			"ok": true,
			"result": {
				"title": "Page Title",
				"content": [
					{"tag": "p", "children": ["text"]},
					{"tag": "img", "attrs": {"src": "http://img1.jpg"}}
				]
			}
		}`
		w.Write([]byte(response))
	}))
	defer ts.Close()

	client := &Client{Token: "t", BaseURL: ts.URL}
	title, images, err := client.GetPage("test-page")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if title != "Page Title" {
		t.Errorf("expected title, got %s", title)
	}
	if len(images) != 1 {
		t.Errorf("expected 1 image")
	}
}

func TestGetPage_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok": false, "error": "PAGE_NOT_FOUND"}`))
	}))
	defer ts.Close()

	client := &Client{BaseURL: ts.URL}
	_, _, err := client.GetPage("invalid")
	if err == nil {
		t.Error("expected error for page not found")
	}
	if err.Error() != "PAGE_NOT_FOUND" {
		t.Errorf("expected PAGE_NOT_FOUND, got %v", err)
	}
}

func TestNetworkErrors(t *testing.T) {
	// Close server immediately to simulate network error
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	client := &Client{Token: "t", BaseURL: ts.URL}

	// CreatePage network error
	res := client.CreatePage("T", nil)
	if !strings.HasPrefix(res, "Ошибка") {
		t.Errorf("expected network error message, got %s", res)
	}

	// EditPage network error
	res = client.EditPage("p", "t", nil, "")
	if !strings.HasPrefix(res, "Ошибка") {
		t.Errorf("expected network error message, got %s", res)
	}

	// GetPage network error
	_, _, err := client.GetPage("p")
	if err == nil {
		t.Error("expected error for GetPage network failure")
	}
}

func TestJsonErrors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{invalid json`))
	}))
	defer ts.Close()
	
	client := &Client{Token: "t", BaseURL: ts.URL}

	// CreatePage JSON error from API
	res := client.CreatePage("T", nil)
	if !strings.HasPrefix(res, "Ошибка") {
		t.Errorf("expected JSON error message, got %s", res)
	}
	
	// GetPage JSON error from API
	_, _, err := client.GetPage("p")
	if err == nil {
		t.Error("expected error for GetPage bad JSON")
	}
}

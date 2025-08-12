package archive

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDeriveFilename(t *testing.T) {
	cases := []struct {
		in   string
		idx  int
		want string
	}{
		{"https://host/a.pdf", 0, "a.pdf"},
		{"https://host/path/", 1, "path"},
		{"   ", 2, "file-3"},
	}
	for _, c := range cases {
		if got := deriveFilename(c.in, c.idx); got != c.want {
			t.Fatalf("deriveFilename(%q,%d)=%q want %q", c.in, c.idx, got, c.want)
		}
	}
}

func newStubServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok.pdf":
			_, _ = io.Copy(w, bytes.NewBufferString("hello"))
		case "/bad":
			http.Error(w, "nope", http.StatusTeapot)
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestBuildArchive_SuccessAndFailures(t *testing.T) {
	srv := newStubServer()
	defer srv.Close()

	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "out.zip")
	ctx := WithHTTPTimeout(context.Background(), 2*time.Second)
	urls := []string{srv.URL + "/ok.pdf", srv.URL + "/bad", srv.URL + "/ok.pdf"}

	results, err := BuildArchive(ctx, dest, urls)
	if err != nil {
		t.Fatalf("BuildArchive error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results[0].Err != "" || results[1].Err == "" || results[2].Err != "" {
		t.Fatalf("unexpected results: %+v", results)
	}
	if results[0].Filename == results[2].Filename {
		t.Fatalf("expected unique filenames in results, got %q and %q", results[0].Filename, results[2].Filename)
	}

	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("zip not created: %v", err)
	}
}

func TestBuildArchive_NoURLs(t *testing.T) {
	_, err := BuildArchive(context.Background(), filepath.Join(t.TempDir(), "x.zip"), nil)
	if err == nil || !strings.Contains(err.Error(), "no urls") {
		t.Fatalf("expected error for no urls, got %v", err)
	}
}

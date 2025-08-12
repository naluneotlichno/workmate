package api

import (
	"archive/zip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"workmate/internal/back/archive"
	"workmate/internal/back/task"

	"context"

	"github.com/gin-gonic/gin"
)

func setupRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	testRouter := gin.Default()
	testManager := task.NewManagerWithOptions(task.Options{DataDir: t.TempDir(), AllowedExtensions: []string{".pdf", ".jpeg"}, MaxConcurrentTasks: 3})
	apiHandler := NewAPI(testManager)
	apiHandler.RegisterRoutes(testRouter)
	return testRouter
}

func TestCreateTask(t *testing.T) {
	testRouter := setupRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["task_id"] == "" {
		t.Fatalf("expected non-empty task_id")
	}

	if resp["status"] != string(task.StatusCreated) {
		t.Fatalf("expected status %q, got %v", task.StatusCreated, resp["status"])
	}
}

func TestAddFilesAndStatus(t *testing.T) {
	testRouter := setupRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	id := resp["task_id"].(string)

	body := `{"urls":["https://example.org/a.pdf","https://example.org/b.jpeg"]}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+id+"/files", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAddThreeFilesTriggersProcessing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testRouter := gin.Default()
	testManager := task.NewManagerWithOptions(task.Options{DataDir: t.TempDir(), AllowedExtensions: []string{".pdf", ".jpeg"}, MaxConcurrentTasks: 3})

	testManager.UseArchiveBuilder(func(ctx context.Context, dest string, urls []string) ([]archive.Result, error) {
		results := make([]archive.Result, len(urls))
		for i := range results {
			results[i].Filename = "f"
			if i == len(results)-1 {
				results[i].Err = "boom"
			}
		}
		return results, nil
	})
	apiHandler := NewAPI(testManager)
	apiHandler.RegisterRoutes(testRouter)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	id := resp["task_id"].(string)

	body := `{"urls":["https://e.org/a.pdf","https://e.org/b.jpeg","https://e.org/c.pdf"]}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+id+"/files", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var addResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &addResp); err != nil {
		t.Fatalf("unmarshal addResp: %v", err)
	}
	if addResp["archive_url"] == nil || addResp["archive_url"].(string) == "" {
		t.Fatalf("expected archive_url to be present when files count is 3")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if tsk, ok := testManager.GetTask(id); ok {
			if tsk.Status == task.StatusReady || tsk.Status == task.StatusFailed {
				if tsk.Status != task.StatusReady {
					t.Fatalf("expected ready, got %s", tsk.Status)
				}
				if tsk.ArchivePath == "" {
					t.Fatalf("expected archive path set")
				}
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for task to be processed")
}

func TestInvalidExtension(t *testing.T) {
	testRouter := setupRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	id := resp["task_id"].(string)

	body := `{"urls":["https://e.org/a.exe"]}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+id+"/files", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTooManyFiles(t *testing.T) {
	testRouter := setupRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	id := resp["task_id"].(string)

	body := `{"urls":["https://e.org/a.pdf","https://e.org/b.jpeg","https://e.org/c.pdf","https://e.org/d.pdf"]}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+id+"/files", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetTaskNotFound(t *testing.T) {
	testRouter := setupRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/does-not-exist", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetTaskOK(t *testing.T) {
	testRouter := setupRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	id := resp["task_id"].(string)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+id, nil)
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var getResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("unmarshal get: %v", err)
	}
	if getResp["id"].(string) != id {
		t.Fatalf("expected id %s, got %v", id, getResp["id"])
	}
}

func TestDownloadArchiveNotReady(t *testing.T) {
	testRouter := setupRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	id := resp["task_id"].(string)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+id+"/archive", nil)
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestDownloadArchiveReady(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testRouter := gin.Default()
	testManager := task.NewManagerWithOptions(task.Options{DataDir: t.TempDir(), AllowedExtensions: []string{".pdf", ".jpeg"}, MaxConcurrentTasks: 3})

	testManager.UseArchiveBuilder(func(ctx context.Context, dest string, urls []string) ([]archive.Result, error) {
		f, err := os.Create(dest)
		if err != nil {
			return nil, err
		}
		zw := zip.NewWriter(f)
		_, _ = zw.Create("ok.txt")
		_ = zw.Close()
		_ = f.Close()
		results := make([]archive.Result, len(urls))
		for i := range results {
			results[i].Filename = "f.pdf"
		}
		return results, nil
	})
	apiHandler := NewAPI(testManager)
	apiHandler.RegisterRoutes(testRouter)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	id := resp["task_id"].(string)

	body := `{"urls":["https://e.org/a.pdf","https://e.org/b.jpeg","https://e.org/c.pdf"]}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+id+"/files", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if tsk, ok := testManager.GetTask(id); ok {
			if tsk.Status == task.StatusReady {
				break
			}
			if tsk.Status == task.StatusFailed {
				t.Fatalf("unexpected failed status")
			}
		}
		time.Sleep(5 * time.Millisecond)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+id+"/archive", nil)
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestServerBusyOnCreate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testRouter := gin.Default()

	testManager := task.NewManagerWithOptions(task.Options{DataDir: t.TempDir(), AllowedExtensions: []string{".pdf", ".jpeg"}, MaxConcurrentTasks: 1})

	blocker := make(chan struct{})
	testManager.UseArchiveBuilder(func(ctx context.Context, dest string, urls []string) ([]archive.Result, error) {
		<-blocker
		return make([]archive.Result, len(urls)), nil
	})
	apiHandler := NewAPI(testManager)
	apiHandler.RegisterRoutes(testRouter)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	id := resp["task_id"].(string)

	body := `{"urls":["https://e.org/a.pdf","https://e.org/b.jpeg","https://e.org/c.pdf"]}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+id+"/files", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil)
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}

	close(blocker)
}

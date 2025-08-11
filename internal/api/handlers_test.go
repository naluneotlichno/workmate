package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"workmate/internal/archive"
	"workmate/internal/task"

	"context"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	testRouter := gin.Default()
	testManager := task.NewManager()
	apiHandler := NewAPI(testManager)
	apiHandler.RegisterRoutes(testRouter)
	return testRouter
}

func TestCreateTask(t *testing.T) {
	testRouter := setupRouter()

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
	testRouter := setupRouter()

	// create task
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

	// add only two files (should not start processing yet, but should accept)
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
	testManager := task.NewManager()
	// inject fake builder: mark two ok and one failed
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

	// create
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

	// add three
	body := `{"urls":["https://e.org/a.pdf","https://e.org/b.jpeg","https://e.org/c.pdf"]}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+id+"/files", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// wait until background processing flips status to ready
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
	testRouter := setupRouter()

	// create task
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	id := resp["task_id"].(string)

	// add one invalid ext
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
	testRouter := setupRouter()

	// create task
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	id := resp["task_id"].(string)

	// add 4 urls
	body := `{"urls":["https://e.org/a.pdf","https://e.org/b.jpeg","https://e.org/c.pdf","https://e.org/d.pdf"]}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+id+"/files", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestServerBusyOnCreate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testRouter := gin.Default()
	// prepare manager with 1 slot to make test tight
	testManager := task.NewManagerWithOptions(task.Options{DataDir: "data", AllowedExtensions: []string{".pdf", ".jpeg"}, MaxConcurrentTasks: 1})
	// inject a builder that blocks until we release it
	blocker := make(chan struct{})
	testManager.UseArchiveBuilder(func(ctx context.Context, dest string, urls []string) ([]archive.Result, error) {
		<-blocker
		return make([]archive.Result, len(urls)), nil
	})
	apiHandler := NewAPI(testManager)
	apiHandler.RegisterRoutes(testRouter)

	// create task 1
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	id := resp["task_id"].(string)

	// fill it with 3 urls to start processing and occupy the only slot
	body := `{"urls":["https://e.org/a.pdf","https://e.org/b.jpeg","https://e.org/c.pdf"]}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+id+"/files", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// try to create second task while first is running (1 slot total) => busy
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil)
	w = httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}

	// unblock and let goroutine finish
	close(blocker)
}

package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"workmate/internal/task"
)

type createTaskResponse struct {
	TaskID string      `json:"task_id"`
	Status task.Status `json:"status"`
}

type addFilesRequest struct {
	URLs []string `json:"urls"`
}

type taskResponse struct {
	ID         string         `json:"id"`
	Status     task.Status    `json:"status"`
	CreatedAt  string         `json:"created_at"`
	Files      []task.FileRef `json:"files"`
	ArchiveURL string         `json:"archive_url,omitempty"`
}

type API struct {
	taskManager *task.Manager
}

const archiveURLFilesThreshold = 3

func NewAPI(taskManager *task.Manager) *API {
	return &API{taskManager: taskManager}
}

// RegisterRoutes registers API routes on the provided gin engine
func (a *API) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		api.POST("/tasks", a.CreateTask)
		api.POST("/tasks/:id/files", a.AddFiles)
		api.GET("/tasks/:id", a.GetTask)
		api.GET("/tasks/:id/archive", a.DownloadArchive)
	}
}

// CreateTask handles creation of a new task
func (a *API) CreateTask(c *gin.Context) {
	if a.taskManager.IsBusy() {
		log.Warn().Msg("rejecting task creation: server is at max concurrency")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "server busy"})
		return
	}
	createdTask := a.taskManager.CreateTask()
	log.Info().Str("task_id", createdTask.ID).Time("created_at", createdTask.CreatedAt).Msg("task created")
	c.JSON(http.StatusCreated, createTaskResponse{TaskID: createdTask.ID, Status: createdTask.Status})
}

// AddFiles attaches up to 3 URLs to the task and triggers processing when full
func (a *API) AddFiles(c *gin.Context) {
	id := c.Param("id")
	var req addFilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn().Str("task_id", id).Err(err).Msg("invalid add files request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	currentTask, err := a.taskManager.AddFiles(id, req.URLs)
	if err != nil {
		if err.Error() == "task not found" {
			log.Warn().Str("task_id", id).Msg("task not found on add files")
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		log.Warn().Str("task_id", id).Err(err).Msg("failed to add files")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log.Info().Str("task_id", currentTask.ID).Int("files_total", len(currentTask.Files)).Msg("files added to task")
	c.JSON(http.StatusOK, a.toTaskResponse(currentTask, c))
}

// GetTask returns task status
func (a *API) GetTask(c *gin.Context) {
	id := c.Param("id")
	if foundTask, ok := a.taskManager.GetTask(id); ok {
		c.JSON(http.StatusOK, a.toTaskResponse(foundTask, c))
		return
	}
	log.Warn().Str("task_id", id).Msg("task not found on get")
	c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
}

// DownloadArchive serves the archive file when ready
func (a *API) DownloadArchive(c *gin.Context) {
	id := c.Param("id")
	foundTask, ok := a.taskManager.GetTask(id)
	if !ok {
		log.Warn().Str("task_id", id).Msg("task not found on download")
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	if foundTask.Status != task.StatusReady || foundTask.ArchivePath == "" {
		log.Warn().Str("task_id", id).Str("status", string(foundTask.Status)).Msg("archive not ready to download")
		c.JSON(http.StatusBadRequest, gin.H{"error": "archive not ready"})
		return
	}
	log.Info().Str("task_id", id).Str("path", foundTask.ArchivePath).Msg("serving archive download")
	c.FileAttachment(foundTask.ArchivePath, "archive-"+foundTask.ID+".zip")
}

func (a *API) toTaskResponse(taskEntity *task.Task, _ *gin.Context) taskResponse {
	resp := taskResponse{
		ID:        taskEntity.ID,
		Status:    taskEntity.Status,
		CreatedAt: taskEntity.CreatedAt.UTC().Format(time.RFC3339),
		Files:     taskEntity.Files,
	}
	// Return archive link as soon as the task has 3 files, even if still processing.
	// The link will become downloadable once status becomes ready.
	if len(taskEntity.Files) >= archiveURLFilesThreshold {
		resp.ArchiveURL = "/api/v1/tasks/" + taskEntity.ID + "/archive"
	}
	return resp
}

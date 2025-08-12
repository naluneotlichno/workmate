package ui

import (
	"embed"
	"html/template"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"workmate/internal/back/task"
)

//go:embed templates/*
var templatesFS embed.FS

type UI struct {
	taskManager *task.Manager
	templates   *template.Template
}

func NewUI(tm *task.Manager) *UI {
	tmpl := template.Must(template.ParseFS(templatesFS, "templates/*.tmpl"))
	return &UI{taskManager: tm, templates: tmpl}
}

func (u *UI) RegisterRoutes(router *gin.Engine) {
	router.SetHTMLTemplate(u.templates)
	router.GET("/", u.UIHome)
	router.GET("/ui/tasks", u.UIOpenExisting)
	router.POST("/ui/tasks", u.UICreateTask)
	router.GET("/ui/tasks/:id", u.UITask)
	router.POST("/ui/tasks/:id/files", u.UIAddFiles)
}

func (u *UI) UIHome(c *gin.Context) { c.HTML(http.StatusOK, "home", gin.H{}) }

func (u *UI) UIOpenExisting(c *gin.Context) {
	id := strings.TrimSpace(c.Query("id"))
	if id == "" {
		c.Redirect(http.StatusFound, "/")
		return
	}
	c.Redirect(http.StatusFound, "/ui/tasks/"+id)
}

func (u *UI) UICreateTask(c *gin.Context) {
	if u.taskManager.IsBusy() {
		c.HTML(http.StatusServiceUnavailable, "home", gin.H{"Error": "server busy: try again later"})
		return
	}
	t := u.taskManager.CreateTask()
	c.Redirect(http.StatusFound, "/ui/tasks/"+t.ID)
}

func (u *UI) UITask(c *gin.Context) {
	id := c.Param("id")
	if t, ok := u.taskManager.GetTask(id); ok {
		c.HTML(http.StatusOK, "task", gin.H{"Task": t, "content": "content-task"})
		return
	}
	c.HTML(http.StatusNotFound, "home", gin.H{"Error": "task not found"})
}

func (u *UI) UIAddFiles(c *gin.Context) {
	id := c.Param("id")
	urls := c.PostFormArray("urls")
	filtered := make([]string, 0, len(urls))
	for _, raw := range urls {
		raw = strings.TrimSpace(raw)
		if raw != "" {
			filtered = append(filtered, raw)
		}
	}
	if len(filtered) > 0 {
		if _, err := u.taskManager.AddFiles(id, filtered); err != nil {
			if t, ok := u.taskManager.GetTask(id); ok {
				c.HTML(http.StatusBadRequest, "task", gin.H{"Task": t, "Error": err.Error()})
				return
			}
			c.HTML(http.StatusBadRequest, "home", gin.H{"Error": err.Error()})
			return
		}
	}
	c.Redirect(http.StatusFound, "/ui/tasks/"+id)
}

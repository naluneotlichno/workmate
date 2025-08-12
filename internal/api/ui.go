package api

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

var uiTemplates = template.Must(template.New("layout").Parse(`{{define "layout"}}
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1"/>
  <title>Workmate UI</title>
  <style>
    body{font-family:system-ui,-apple-system,Segoe UI,Roboto,Ubuntu,Cantarell,Noto Sans,sans-serif;max-width:880px;margin:32px auto;padding:0 16px;color:#0b0b0b;background:#fafafa}
    header{margin-bottom:24px}
    h1{font-size:22px;margin:0 0 8px}
    a{color:#0b63e5;text-decoration:none}
    a:hover{text-decoration:underline}
    .card{background:#fff;border:1px solid #e9e9e9;border-radius:10px;padding:16px;margin:12px 0}
    .row{display:flex;gap:12px;flex-wrap:wrap}
    .btn{display:inline-block;background:#0b63e5;color:#fff;border:none;padding:10px 14px;border-radius:8px;cursor:pointer}
    .btn.secondary{background:#444}
    input[type=text]{padding:9px 10px;border:1px solid #dcdcdc;border-radius:8px;width:100%}
    .muted{color:#666}
    .mono{font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace}
    .grid{display:grid;grid-template-columns:1fr 1fr;gap:12px}
    .list{margin:0;padding-left:18px}
    .status{display:inline-block;padding:4px 8px;border-radius:6px;background:#efefef;font-size:12px}
    footer{margin-top:24px;color:#666;font-size:12px}
  </style>
  </head>
<body>
  <header>
    <h1>Workmate UI</h1>
    <div class="muted">Minimal no-JS helper for API</div>
  </header>
  {{template "content" .}}
  <footer>
    <div>API base: <span class="mono">/api/v1</span></div>
  </footer>
</body>
</html>
{{end}}

{{define "home"}}
  {{template "layout" .}}
{{end}}

{{define "content"}}
  {{if .Error}}
  <div class="card" style="border-color:#f2b8b5;background:#fff6f6">
    <strong style="color:#b3261e">Error:</strong> <span class="muted">{{.Error}}</span>
  </div>
  {{end}}
  <div class="card">
    <h2>Create task</h2>
    <form method="post" action="/ui/tasks">
      <button class="btn" type="submit">Create</button>
    </form>
    <div class="muted">POST /api/v1/tasks</div>
  </div>

  <div class="card">
    <h2>Open existing task</h2>
    <form method="get" action="/ui/tasks">
      <div class="row">
        <input type="text" name="id" placeholder="Task ID" required />
        <button class="btn" type="submit">Open</button>
      </div>
    </form>
    <div class="muted">GET /api/v1/tasks/{id}</div>
  </div>

  <div class="card">
    <h2>API quick links</h2>
    <ul class="list">
      <li><a href="/api/v1/tasks" target="_blank">POST /api/v1/tasks</a> (open to see JSON in new tab)</li>
      <li class="muted">Other endpoints require task id: <span class="mono">/api/v1/tasks/{id}/files</span>, <span class="mono">/api/v1/tasks/{id}</span>, <span class="mono">/api/v1/tasks/{id}/archive</span></li>
    </ul>
  </div>
{{end}}

{{define "layout_task"}}
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1"/>
  <title>Workmate UI · Task</title>
  <style>
    body{font-family:system-ui,-apple-system,Segoe UI,Roboto,Ubuntu,Cantarell,Noto Sans,sans-serif;max-width:880px;margin:32px auto;padding:0 16px;color:#0b0b0b;background:#fafafa}
    header{margin-bottom:24px}
    h1{font-size:22px;margin:0 0 8px}
    a{color:#0b63e5;text-decoration:none}
    a:hover{text-decoration:underline}
    .card{background:#fff;border:1px solid #e9e9e9;border-radius:10px;padding:16px;margin:12px 0}
    .row{display:flex;gap:12px;flex-wrap:wrap}
    .btn{display:inline-block;background:#0b63e5;color:#fff;border:none;padding:10px 14px;border-radius:8px;cursor:pointer}
    .btn.secondary{background:#444}
    input[type=text]{padding:9px 10px;border:1px solid #dcdcdc;border-radius:8px;width:100%}
    .muted{color:#666}
    .mono{font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace}
    .grid{display:grid;grid-template-columns:1fr 1fr;gap:12px}
    .list{margin:0;padding-left:18px}
    .status{display:inline-block;padding:4px 8px;border-radius:6px;background:#efefef;font-size:12px}
    footer{margin-top:24px;color:#666;font-size:12px}
  </style>
  </head>
<body>
  <header>
    <h1><a href="/">Workmate UI</a></h1>
    <div class="muted">Minimal no-JS helper for API</div>
  </header>
  {{template "content-task" .}}
  <footer>
    <div>API base: <span class="mono">/api/v1</span></div>
  </footer>
</body>
</html>
{{end}}

{{define "task"}}
  {{template "layout_task" .}}
{{end}}

{{define "content-task"}}
  {{if .Error}}
  <div class="card" style="border-color:#f2b8b5;background:#fff6f6">
    <strong style="color:#b3261e">Error:</strong> <span class="muted">{{.Error}}</span>
  </div>
  {{end}}
  <div class="card">
    <h2>Task <span class="mono" id="taskId">{{.Task.ID}}</span></h2>
    {{if .Task.Title}}
    <div>Title: <strong id="taskTitle">{{.Task.Title}}</strong></div>
    {{end}}
    <div>Status: <span class="status" id="taskStatus">{{.Task.Status}}</span></div>
    <div class="muted">Created at: <span id="taskCreatedAt">{{.Task.CreatedAt}}</span></div>
  </div>

  <div class="card">
    <h3>Files</h3>
    <ul class="list" id="filesList">
      {{if .Task.Files}}
        {{range .Task.Files}}
          <li>
            <div><span class="mono">{{.URL}}</span></div>
            <div class="muted">{{.State}}{{if .Filename}} · {{.Filename}}{{end}}{{if .Error}} · error: {{.Error}}{{end}}</div>
          </li>
        {{end}}
      {{end}}
    </ul>
    {{if not .Task.Files}}
      <div class="muted" id="noFilesHint">No files yet</div>
    {{end}}
  </div>

  <div class="card">
    <h3>Add up to 3 URLs (.pdf, .jpeg)</h3>
    <form method="post" action="/ui/tasks/{{.Task.ID}}/files">
      <div class="grid">
        <input type="text" name="urls" placeholder="https://host/a.pdf" />
        <input type="text" name="urls" placeholder="https://host/b.jpeg" />
        <input type="text" name="urls" placeholder="https://host/c.pdf" />
      </div>
      <div style="margin-top:12px"><button class="btn" type="submit">Add</button>
        <a class="btn secondary" href="/ui/tasks/{{.Task.ID}}" style="margin-left:8px">Refresh</a>
      </div>
    </form>
    <div class="muted">POST /api/v1/tasks/{{.Task.ID}}/files</div>
  </div>

  <div class="card">
    <h3>Archive</h3>
    <div>
      <a class="btn" id="downloadBtn" href="/api/v1/tasks/{{.Task.ID}}/archive">Download zip</a>
      <span class="muted" style="margin-left:8px">Link works when status is ready</span>
    </div>
    <div class="muted">GET /api/v1/tasks/{{.Task.ID}}/archive</div>
  </div>

  <script>
  (function() {
    const taskId = document.getElementById('taskId').textContent;
    const statusEl = document.getElementById('taskStatus');
    const titleEl = document.getElementById('taskTitle');
    const createdAtEl = document.getElementById('taskCreatedAt');
    const filesListEl = document.getElementById('filesList');
    const noFilesHintEl = document.getElementById('noFilesHint');
    const downloadBtn = document.getElementById('downloadBtn');

    function setDownloadEnabled(enabled) {
      if (!downloadBtn) return;
      if (enabled) {
        downloadBtn.style.pointerEvents = 'auto';
        downloadBtn.style.opacity = '1';
        downloadBtn.setAttribute('aria-disabled', 'false');
      } else {
        downloadBtn.style.pointerEvents = 'none';
        downloadBtn.style.opacity = '0.6';
        downloadBtn.setAttribute('aria-disabled', 'true');
      }
    }

    async function refreshTask() {
      try {
        const res = await fetch('/api/v1/tasks/' + encodeURIComponent(taskId), { headers: { 'Accept': 'application/json' } });
        if (!res.ok) return;
        const data = await res.json();

        if (data.status && statusEl) statusEl.textContent = data.status;
        if (data.title && titleEl) titleEl.textContent = data.title;
        if (data.created_at && createdAtEl) createdAtEl.textContent = data.created_at;

        if (Array.isArray(data.files) && filesListEl) {
          filesListEl.innerHTML = '';
          if (data.files.length === 0 && noFilesHintEl) {
            noFilesHintEl.style.display = 'block';
          } else if (noFilesHintEl) {
            noFilesHintEl.style.display = 'none';
          }
          for (const f of data.files) {
            const li = document.createElement('li');
            const urlDiv = document.createElement('div');
            urlDiv.innerHTML = '<span class="mono"></span>';
            urlDiv.querySelector('span').textContent = f.url || '';
            const metaDiv = document.createElement('div');
            metaDiv.className = 'muted';
            const parts = [f.state || ''];
            if (f.filename) parts.push('· ' + f.filename);
            if (f.error) parts.push('· error: ' + f.error);
            metaDiv.textContent = parts.join(' ');
            li.appendChild(urlDiv);
            li.appendChild(metaDiv);
            filesListEl.appendChild(li);
          }
        }

        const ready = data.status === 'ready' && !!data.archive_url;
        setDownloadEnabled(ready);

        // Stop polling when terminal state reached
        if (data.status === 'ready' || data.status === 'failed') {
          clearInterval(timerId);
        }
      } catch (_) {
        // Ignore network errors while polling
      }
    }

    // Initial state
    setDownloadEnabled(false);
    const timerId = setInterval(refreshTask, 1500);
    refreshTask();
  })();
  </script>
{{end}}
`))

func (a *API) RegisterUIRoutes(router *gin.Engine) {
	router.SetHTMLTemplate(uiTemplates)
	router.GET("/", a.UIHome)
	router.GET("/ui/tasks", a.UIOpenExisting)
	router.POST("/ui/tasks", a.UICreateTask)
	router.GET("/ui/tasks/:id", a.UITask)
	router.POST("/ui/tasks/:id/files", a.UIAddFiles)
}

func (a *API) UIHome(c *gin.Context) { c.HTML(http.StatusOK, "home", gin.H{}) }

func (a *API) UIOpenExisting(c *gin.Context) {
	id := strings.TrimSpace(c.Query("id"))
	if id == "" {
		c.Redirect(http.StatusFound, "/")
		return
	}
	c.Redirect(http.StatusFound, "/ui/tasks/"+id)
}

func (a *API) UICreateTask(c *gin.Context) {
	if a.taskManager.IsBusy() {
		c.HTML(http.StatusServiceUnavailable, "home", gin.H{"Error": "server busy: try again later"})
		return
	}
	t := a.taskManager.CreateTask()
	c.Redirect(http.StatusFound, "/ui/tasks/"+t.ID)
}

func (a *API) UITask(c *gin.Context) {
	id := c.Param("id")
	if t, ok := a.taskManager.GetTask(id); ok {
		c.HTML(http.StatusOK, "task", gin.H{"Task": t, "content": "content-task"})

		return
	}
	c.HTML(http.StatusNotFound, "home", gin.H{"Error": "task not found"})
}

func (a *API) UIAddFiles(c *gin.Context) {
	id := c.Param("id")
	urls := c.PostFormArray("urls")
	filtered := make([]string, 0, len(urls))
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u != "" {
			filtered = append(filtered, u)
		}
	}
	if len(filtered) > 0 {
		if _, err := a.taskManager.AddFiles(id, filtered); err != nil {

			if t, ok := a.taskManager.GetTask(id); ok {
				c.HTML(http.StatusBadRequest, "task", gin.H{"Task": t, "Error": err.Error()})
				return
			}
			c.HTML(http.StatusBadRequest, "home", gin.H{"Error": err.Error()})
			return
		}
	}
	c.Redirect(http.StatusFound, "/ui/tasks/"+id)
}

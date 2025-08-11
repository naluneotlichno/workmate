Workmate — simple archive builder (no DB, no Docker)

Overview

- Embedded service that downloads up to 3 files per task (.pdf, .jpeg), zips them, and returns the archive link.
- No external infra: in-memory state + JSON snapshots on disk.
- Concurrency limit: up to 3 tasks processed in parallel.

Run

- make run
- Server listens on :8080

Config (optional config.yml)

```yaml
port: 8080
data_dir: data
allowed_extensions: [".pdf", ".jpeg"]
max_concurrent_tasks: 3
```

Storage layout

```
data/
  tasks/
    <task_id>/
      status.json
      archive.zip   # appears when ready
```

API

- POST /api/v1/tasks

  - 201 {"task_id":"...","status":"created"}
  - 503 {"error":"server busy"}

- POST /api/v1/tasks/{id}/files

  - body: {"urls":["https://.../a.pdf","https://.../b.jpeg","https://.../c.pdf"]}
  - 200 task status JSON (includes files, and archive_url once ready)

- GET /api/v1/tasks/{id}

  - 200 task status JSON

- GET /api/v1/tasks/{id}/archive
  - 200 zip attachment if ready, else 400

Curl examples

```bash
curl -X POST http://localhost:8080/api/v1/tasks

curl -X POST http://localhost:8080/api/v1/tasks/<id>/files \
  -H 'Content-Type: application/json' \
  -d '{"urls":["https://host/a.pdf","https://host/b.jpeg","https://host/c.pdf"]}'

curl http://localhost:8080/api/v1/tasks/<id>

curl -OJ http://localhost:8080/api/v1/tasks/<id>/archive
```

Notes

- On restart, tasks that were in_progress become failed; snapshots are reloaded into memory.
- Partial failures do not block archive creation: failed files are marked, ok files are zipped.

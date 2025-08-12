Workmate — простой сборщик архивов (без БД и без Docker)

Обзор

- Встроенный сервис, который скачивает до 3 файлов на задачу (.pdf, .jpeg), упаковывает их в zip и возвращает ссылку на архив.
- Без внешней инфраструктуры: состояние в памяти + JSON-снимки на диске.
- Ограничение параллелизма: одновременно обрабатывается до 3 задач.

Запуск

- make up — сборка + линтеры + тесты + запуск бинарника в фоне. Сервер слушает :8080
- make down — корректная остановка и очистка бинарника
- make clean — удаление артефактов сборки
- make test — запуск тестов Go
- make lint — форматирование (go fmt) и запуск линтеров (golangci-lint)
- make build — сборка бинарника в `bin/workmate.exe`

Предварительные требования

- Go 1.22+
- Make (на Windows работает через Git Bash)
- golangci-lint (опционально): `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`

Конфигурация (опционально `config.yml`)

```yaml
port: 8080
data_dir: data
allowed_extensions: [".pdf", ".jpeg"]
max_concurrent_tasks: 3
```

Структура хранилища

```
data/
  tasks/
    <task_id>/
      status.json
      archive.zip   # появляется, когда архив готов
```

Как запускать коротко

1. `make up` — всё поднимет и запустит. Потом:
   - UI: открой `http://localhost:8080`
   - API: см. раздел API ниже (curl examples)
2. Остановить: `make down`
3. Dev-loop: `make lint && make test && make build`

API

- POST /api/v1/tasks

  - 201 {"task_id":"...","status":"created"}
  - 503 {"error":"server busy"} (сервер занят, достигнут лимит параллелизма)

- POST /api/v1/tasks/{id}/files

  - тело запроса: {"urls":["https://.../a.pdf","https://.../b.jpeg","https://.../c.pdf"]}
  - 200 — JSON статуса задачи (содержит файлы, а когда архив готов — `archive_url`)

- GET /api/v1/tasks/{id}

  - 200 — JSON статуса задачи

- GET /api/v1/tasks/{id}/archive
  - 200 — скачивание zip, если архив готов; иначе 400

Примеры curl

```bash
curl -X POST http://localhost:8080/api/v1/tasks

curl -X POST http://localhost:8080/api/v1/tasks/<id>/files \
  -H 'Content-Type: application/json' \
  -d '{"urls":["https://host/a.pdf","https://host/b.jpeg","https://host/c.pdf"]}'

curl http://localhost:8080/api/v1/tasks/<id>

curl -OJ http://localhost:8080/api/v1/tasks/<id>/archive
```

Примечания

- При рестарте задачи со статусом `in_progress` помечаются как `failed`; снимки состояния загружаются обратно в память.
- Частичные ошибки не блокируют создание архива: неуспешные файлы помечаются, успешные упаковываются в архив.

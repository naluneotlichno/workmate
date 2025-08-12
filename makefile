SHELL := bash

.PHONY: build run stop clean up down gmt lint test run-debug

APP_NAME := workmate
BIN_DIR := storage
BIN := $(BIN_DIR)/$(APP_NAME).exe
MAIN := ./cmd/main.go
PID_FILE := $(BIN_DIR)/temp_pid.txt
TEST_DIRS ?= ./...
PORT ?= 8080


up: down gmt lint test build
	@echo "[run-debug] Launching $(BIN) with visible window..."
	@if command -v powershell >/dev/null 2>&1; then \
	  powershell -NoProfile -Command '$$p = Start-Process -FilePath "$(BIN)" -WindowStyle Normal -PassThru; Set-Content -Path "$(PID_FILE)" -Value $$p.Id -Encoding ascii'; \
	elif command -v pwsh >/dev/null 2>&1; then \
	  pwsh -NoProfile -Command '$$p = Start-Process -FilePath "$(BIN)" -WindowStyle Normal -PassThru; Set-Content -Path "$(PID_FILE)" -Value $$p.Id -Encoding ascii'; \
	else \
	  "$(BIN)" & echo $$! > "$(PID_FILE)"; \
	fi
	@echo "[run-debug] PID saved to $(PID_FILE): $$(cat "$(PID_FILE)")"

gmt:
	@go mod tidy

lint: 
	@go fmt ./...
	@echo "Start linters..."
	@golangci-lint run --timeout=10m ./... && echo "All linters have run successfully"

test:
	@echo "Running tests..."
	@go test -v $(TEST_DIRS)

build:
	@echo "[build] Starting build..."
	@mkdir -p $(BIN_DIR)
	@go build -o "$(BIN)" $(MAIN)
	@echo "[build] Binary built at $(BIN)"

down:
	@echo "[stop] Stopping application..."
	@if [ -f "$(PID_FILE)" ]; then \
	  PID=$$(cat "$(PID_FILE)"); \
	  echo "[stop] Found PID $$PID"; \
      if command -v powershell >/dev/null 2>&1; then \
        echo "[stop] Using PowerShell to terminate PID $$PID"; \
        powershell -NoProfile -Command "Stop-Process -Id $$PID -Force -ErrorAction SilentlyContinue" >/dev/null 2>&1 || true; \
      elif command -v taskkill >/dev/null 2>&1; then \
        echo "[stop] Using taskkill to terminate PID $$PID"; \
        taskkill /PID $$PID /F /T >/dev/null 2>&1 || true; \
	  else \
	    echo "[stop] Using kill -SIGINT for PID $$PID"; \
	    kill -SIGINT $$PID >/dev/null 2>&1 || true; \
	  fi; \
	  rm -f "$(PID_FILE)"; \
	  echo "[stop] PID file removed"; \
	else \
	  echo "[stop] No $(PID_FILE) found; nothing to stop"; \
	fi
	# Fallback: kill by image name (best-effort; covers manual runs)
	@if command -v powershell >/dev/null 2>&1; then \
	  echo "[stop] Killing all $(APP_NAME).exe via PowerShell (best-effort)"; \
	  powershell -NoProfile -Command "Get-Process -Name '$(APP_NAME)' -ErrorAction SilentlyContinue | Stop-Process -Force -PassThru | Out-Null" >/dev/null 2>&1 || true; \
	elif command -v taskkill >/dev/null 2>&1; then \
	  echo "[stop] Killing all $(APP_NAME).exe via taskkill (best-effort)"; \
	  taskkill /IM $(APP_NAME).exe /F /T >/dev/null 2>&1 || true; \
	elif command -v powershell >/dev/null 2>&1; then \
	  echo "[stop] Killing by PowerShell Get-Process (best-effort)"; \
	  powershell -NoProfile -Command "Get-Process -Name '$(APP_NAME)' -ErrorAction SilentlyContinue | Stop-Process -Force -PassThru | Out-Null" >/dev/null 2>&1 || true; \
	elif command -v pkill >/dev/null 2>&1; then \
	  echo "[stop] Killing by name $(APP_NAME) via pkill (best-effort)"; \
	  pkill -f $(APP_NAME) >/dev/null 2>&1 || true; \
	fi
	@if [ -f "$(BIN)" ]; then \
	  echo "[stop] Removing binary $(BIN)"; \
	  rm -f "$(BIN)"; \
	else \
	  echo "[stop] Binary not found; skip delete"; \
	fi
	# Kill by TCP port as a last resort (Windows PowerShell)
	@if command -v powershell >/dev/null 2>&1; then \
	  echo "[stop] Killing by TCP port $(PORT) via PowerShell (best-effort)"; \
	  powershell -NoProfile -Command "try { $conns = Get-NetTCPConnection -LocalPort $(PORT) -ErrorAction SilentlyContinue; if ($conns) { $pids = $conns | Select-Object -ExpandProperty OwningProcess -Unique; foreach ($pid in $pids) { Stop-Process -Id $pid -Force -ErrorAction SilentlyContinue } } } catch {}" >/dev/null 2>&1 || true; \
	fi
	@echo "[stop] Done"



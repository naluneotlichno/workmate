SHELL := bash

.PHONY: build run stop clean up down gmt lint test

APP_NAME := workmate
BIN_DIR := bin
BIN := $(BIN_DIR)/$(APP_NAME).exe
MAIN := ./cmd/main.go
PID_FILE := temp_pid.txt
TEST_DIRS ?= ./...


up: down gmt lint test build
	@echo "[run] Launching $(BIN) in background..."
	@if command -v powershell >/dev/null 2>&1; then \
	  powershell -NoProfile -Command '$$p = Start-Process -FilePath "$(BIN)" -PassThru; Set-Content -Path "$(PID_FILE)" -Value $$p.Id -Encoding ascii'; \
	elif command -v pwsh >/dev/null 2>&1; then \
	  pwsh -NoProfile -Command '$$p = Start-Process -FilePath "$(BIN)" -PassThru; Set-Content -Path "$(PID_FILE)" -Value $$p.Id -Encoding ascii'; \
	else \
	  "$(BIN)" & echo $$! > "$(PID_FILE)"; \
	fi
	@echo "[run] PID saved to $(PID_FILE): $$(cat "$(PID_FILE)")"

gmt:
	go mod tidy

lint: 
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
      if command -v taskkill >/dev/null 2>&1; then \
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
	@if [ -f "$(BIN)" ]; then \
	  echo "[stop] Removing binary $(BIN)"; \
	  rm -f "$(BIN)"; \
	else \
	  echo "[stop] Binary not found; skip delete"; \
	fi
	@echo "[stop] Done"

clean:
	@echo "[clean] Removing build artifacts..."
	@rm -f $(PID_FILE)
	@rm -rf $(BIN_DIR)
	@echo "[clean] Clean complete"



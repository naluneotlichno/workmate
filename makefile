# Makefile

# Переменная для указания каталогов, где будут выполняться тесты
TEST_DIRS ?= ./...

.PHONY: lint lint-setup migrate-up migrate-down build build-docker run-local mockgen test run-bot build-bot test-bot lint-chain lint-bot run-all-tables clean mock mock mock-clean mock-all mock-clean-all

# ==========================================
# Common Commands
# ==========================================

format:
	@echo "Fixing file endings (CRLF -> LF) for all Go files in the project..."
	@find . -name "*.go" -type f -exec dos2unix {} \; 2>/dev/null || echo "dos2unix not found, skipping line ending conversion"
	@echo "Formatting all Go files in the project..."
	@gofmt -l -w .
	@echo "All files formatted successfully!"

gmt:
	go mod tidy

run: gmt
	go run ./cmd/main.go

# ==========================================
# Chain Service Commands
# ==========================================

test:
	@echo "Running tests..."
	@go test -v $(TEST_DIRS)

lint: gmt format
	@echo "Running go fmt..."
	@go fmt ./... 
	@echo "Start linters..."
	@golangci-lint run --timeout=10m ./... && echo "All linters have run successfully"

# ==========================================
# Mock Generation Commands
# ==========================================

# Команда для установки mockgen, если он не установлен
install-mockgen:
	@which mockgen > /dev/null || (go install go.uber.org/mock/mockgen@latest && export PATH=$$PATH:$$(go env GOPATH)/bin)
	@echo "mockgen installed successfully"

# Команда для генерации моков для конкретного интерфейса
mock:
	@echo "Usage: make mock INTERFACE=path/to/interface.go"
	@echo "Example: make mock INTERFACE=internal/repo/repository.go"

# Команда для генерации мока конкретного интерфейса
mock-interface: install-mockgen
	@if [ -z "$(INTERFACE)" ]; then \
		echo "Error: INTERFACE is not set. Usage: make mock-interface INTERFACE=path/to/interface.go"; \
		exit 1; \
	fi
	@echo "Generating mock for $(INTERFACE)..."
	@mkdir -p internal/mocks
	@mockgen -source=$(INTERFACE) \
		-destination=internal/mocks/$(shell basename $(INTERFACE) .go)_mocks.go \
		-package=mocks
	@echo "Mock generated successfully"

# Команда для генерации всех моков
mock-all: install-mockgen
	@echo "Generating all mocks..."
	@mkdir -p internal/mocks
	@# Генерация моков для репозитория
	mockgen -source=internal/repo/repository.go \
		-destination=internal/mocks/repo_mocks.go -package=mocks
	@# Генерация моков для use case
	mockgen -source=internal/usecase/usecase.go \
		-destination=internal/mocks/usecase_mocks.go -package=mocks
	@# Генерация моков для дополнительных use cases
	mockgen -source=internal/usecase/tgbot_users.go \
		-destination=internal/mocks/users_mocks.go -package=mocks
	mockgen -source=internal/usecase/tgbot_rating.go \
		-destination=internal/mocks/rating_mocks.go -package=mocks
	@# Генерация моков для MinIO
	mockgen -source=internal/repo/minio/interfaces.go \
		-destination=internal/mocks/minio_mocks.go -package=mocks 
	mockgen -source=internal/client/minio/client.go \
		-destination=internal/mocks/minio_client_mocks.go -package=mocks
	@# Генерация моков для Telegram бота
	mockgen -source=internal/infrastructure/tgbot/interfaces.go \
		-destination=internal/mocks/tgbot_mocks.go -package=mocks
	@echo "All mocks generated successfully"
	go mod tidy

# Команда для очистки и пересоздания всех моков
mock-clean-all: 
	@echo "Cleaning all mocks..."
	@rm -rf internal/mocks
	@mkdir -p internal/mocks
	@echo "Old mocks cleaned. Generating new mocks..."
	@make mock-all

# Команда для очистки конкретного мока
mock-clean:
	@if [ -z "$(INTERFACE)" ]; then \
		echo "Error: INTERFACE is not set. Usage: make mock-clean INTERFACE=path/to/interface.go"; \
		exit 1; \
	fi
	@echo "Cleaning mock for $(INTERFACE)..."
	@rm -f internal/mocks/$(shell basename $(INTERFACE) .go)_mocks.go
	@echo "Mock cleaned. Generating new mock..."
	@make mock-interface INTERFACE=$(INTERFACE)

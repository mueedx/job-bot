# Convenience targets for local development. Docker Compose is the recommended path.
.PHONY: help setup dev-api dev-web test test-api test-web build docker docker-down logs seed tidy clean sec sources-check

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

setup: ## Create .env and seed editable config from the examples
	@test -f .env || cp .env.example .env
	@test -f data/target_companies.yaml || cp data/target_companies.yaml.example data/target_companies.yaml
	@echo "Next: add resume PDFs to resumes/ (see resumes/README.md)"

dev-api: ## Run the API locally on :8000
	cd backend && go run ./cmd/server

dev-web: ## Run the dashboard locally on :3000
	cd frontend && npm run dev

test: test-api test-web ## Run every test

test-api: ## Build, vet and test the backend
	cd backend && go build ./... && go vet ./... && go test ./...

test-web: ## Lint and build the frontend
	cd frontend && npm run lint && npm run typecheck && npm run build

sec: ## Scan the repo for leaked secrets (requires gitleaks)
	gitleaks detect --source . --redact

sources-check: ## Probe every enabled job source and report what actually came back
	curl -fsS http://localhost:8000/api/sources/health | jq -r '.results[] | if .checked then "\(.label): \(.count) jobs\(if .ok then "" else " — " + (.error // "no jobs") end)" else "\(.label): skipped (\(.reason))" end'

build: ## Build the Docker images
	docker compose build

docker: ## Run the full stack with Docker Compose
	docker compose up --build

docker-down: ## Stop the stack
	docker compose down

logs: ## Follow API logs
	docker compose logs -f api

seed: ## Insert demo jobs into the running API
	curl -fsS -X POST http://localhost:8000/api/dev/seed && echo

tidy: ## Tidy Go modules
	cd backend && go mod tidy

clean: ## Drop build caches and the local database
	cd backend && go clean -cache -testcache
	rm -f data/jobs.db

# Local development. Needs Docker, Go and Node (see README.md). POSIX make; recipes run with $(SHELL).
SHELL = /bin/sh
.PHONY: dev db seed api web test e2e

# Overridable: make dev UNICOURSE_BASE_URL=...
export UNICOURSE_DATABASE_URL ?= postgres://unicourse:unicourse@localhost:5432/unicourse?sslmode=disable
export UNICOURSE_OWNER_DATABASE_URL ?= postgres://unicourse_owner:unicourse_owner@localhost:5432/unicourse?sslmode=disable
export UNICOURSE_BASE_URL ?= http://localhost:8080
export UNICOURSE_WEB_ORIGIN ?= http://localhost:5173
export UNICOURSE_SESSION_SECRET ?= dev-secret-dev-secret-dev-secret-00
export UNICOURSE_FILES_DIR ?= $(CURDIR)/apps/api/data/files

# Postgres, migrations, the demo university, then the API (:8080) and Vite (:5173) until Ctrl-C.
# Ctrl-C stops both servers; Postgres keeps running: docker compose -p unicourse -f apps/api/docker-compose.yml down
dev: db seed
	@echo "API on $(UNICOURSE_BASE_URL), web on $(UNICOURSE_WEB_ORIGIN); Ctrl-C stops both. Sign-in links appear in the API log."
	@trap 'kill 0' INT TERM; \
	(cd apps/api && exec go run ./cmd/api) & \
	(cd apps/web && exec npm run dev) & \
	wait

db:
	docker compose -p unicourse -f $(CURDIR)/apps/api/docker-compose.yml up -d --wait

# Migrate, then replace the demo university (local databases only).
seed:
	mkdir -p $(UNICOURSE_FILES_DIR)
	cd apps/api && go run ./cmd/unicourse-admin migrate && go run ./cmd/unicourse-admin seed-demo --reset

api:
	cd apps/api && go run ./cmd/api

web:
	cd apps/web && npm run dev

# The test URLs below are the local Postgres from apps/api/docker-compose.yml.
test: export UNICOURSE_TEST_DATABASE_URL ?= postgres://unicourse:unicourse@localhost:5432/unicourse_test?sslmode=disable
test: export UNICOURSE_TEST_OWNER_DATABASE_URL ?= postgres://unicourse_owner:unicourse_owner@localhost:5432/unicourse_test?sslmode=disable
test:
	cd apps/api && gofmt -l . && go vet ./... && go test ./... -count=1 -race
	cd apps/web && npm run lint && npm run typecheck && npm test -- --run

e2e:
	cd apps/web && npm run e2e

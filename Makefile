GIT_REF ?= $(shell git symbolic-ref HEAD)
GIT_REF_NAME ?= $(shell git branch --show-current)
GIT_REF_TYPE ?= branch
GIT_COMMIT ?= $(shell git rev-parse HEAD)
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
WT_OUTPUT_FILE ?= tmp/workout-tracker
WT_DEBUG_OUTPUT_FILE ?= tmp/wt-debug

GO_TEST = go test -short -count 1 -mod vendor -covermode=atomic

BRANCH_NAME_DEPS ?= update-deps

.PHONY: all clean test build meta install-deps

all: clean install-deps test build

release-patch release-minor release-major:
	$(MAKE) release VERSION=$(shell go run github.com/mdomke/git-semver/v6@latest -target $(subst release-,,$@))

release:
	git tag -s -a $(VERSION) -m "Release $(VERSION)"
	@echo "Now run:"
	@echo "- git push --tags"
	@echo "- gh release create --generate-notes $(VERSION)"

install-deps:
	cd client && npm ci

clean:
	rm -fv ./assets/output.css ./workout-tracker
	rm -rf ./tmp/ ./node_modules/ ./assets/dist/

watch/server:
	go run github.com/air-verse/air@latest \
			--build.full_bin           "APP_ENV=development $(WT_OUTPUT_FILE)" \
			--build.cmd                "make build-server" \
			--build.delay              1000 \
			--build.exclude_dir        "assets,client,docs,testdata,tmp,vendor" \
			--build.exclude_regex      "_test.go" \
			--build.exclude_unchanged  false \
			--build.include_ext        "go,html,json,yaml" \
			--build.stop_on_error      true \
			--screen.clear_on_rebuild  false

watch/client: install-deps
	cd client && npm run start

dev-backend:
	$(MAKE) watch/server

dev: dev-postgres

dev-postgres:
	docker compose \
			--project-directory ./docker/ \
			--file ./docker/docker-compose.dev.yaml \
			up --build

dev-activitypub:
	docker compose \
			--project-directory ./docker/ \
			--file ./docker/docker-compose.activitypub.yaml \
			up --build

dev-clean:
	docker compose --project-directory ./docker/ --file ./docker/docker-compose.dev.yaml down --remove-orphans --volumes
	docker compose --project-directory ./docker/ --file ./docker/docker-compose.activitypub.yaml down --remove-orphans --volumes
	docker compose --project-directory ./docker/ --file ./docker/docker-compose.yaml down --remove-orphans --volumes

build: build-client build-server build-image
meta: swagger changelog

build-cli:
	go build \
			-ldflags "-X 'main.buildTime=$(BUILD_TIME)' -X 'main.gitCommit=$(GIT_COMMIT)' -X 'main.gitRef=$(GIT_REF)' -X 'main.gitRefName=$(GIT_REF_NAME)' -X 'main.gitRefType=$(GIT_REF_TYPE)'" \
			-o $(WT_DEBUG_OUTPUT_FILE) ./cmd/wt-debug/

build-server:
	go build \
			-ldflags "-X 'main.buildTime=$(BUILD_TIME)' -X 'main.gitCommit=$(GIT_COMMIT)' -X 'main.gitRef=$(GIT_REF)' -X 'main.gitRefName=$(GIT_REF_NAME)' -X 'main.gitRefType=$(GIT_REF_TYPE)'" \
			-o $(WT_OUTPUT_FILE) ./cmd/workout-tracker/

build-client: install-deps
	cd client && npm run build

build-image:
	docker build \
			--tag workout-tracker --pull \
			--build-arg BUILD_TIME="$(BUILD_TIME)" \
			--build-arg GIT_COMMIT="$(GIT_COMMIT)" \
			--build-arg GIT_REF="$(GIT_REF)" \
			--build-arg GIT_REF_NAME="$(GIT_REF_NAME)" \
			--build-arg GIT_REF_TYPE="$(GIT_REF_TYPE)" \
			--file ./docker/Dockerfile.prod \
			.

swagger:
	go run github.com/swaggo/swag/cmd/swag@latest init \
			--parseDependency \
			--dir ./pkg/app/,./pkg/controller/,./pkg/model/,./pkg/model/dto/,./vendor/gorm.io/gorm/,./vendor/github.com/codingsince1985/geo-golang/ \
			--generalInfo routes.go

generate-workout-types:
	go generate ./...
	node scripts/generate-workout-types.js

test-packages:
	$(GO_TEST) ./pkg/...

test-commands:
	$(GO_TEST) ./cmd/...

serve:
	$(WT_OUTPUT_FILE)

test: test-go test-assets

test-assets:
	prettier --check .

test-go: test-commands test-packages
	golangci-lint run --allow-parallel-runners

go-cover:
	go test -short -count 1 -mod vendor -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	rm -vf coverage.out

update-deps:
	# Check no changes
	@if [[ "$$(git status --porcelain | wc -l)" -gt 0 ]]; then echo "There are changes; please commit or stash them first"; exit 1; fi
	# Check if branch exists locally or remotely
	@if git show-ref --verify --quiet refs/heads/$(BRANCH_NAME_DEPS); then echo "Branch $(BRANCH_NAME_DEPS) already exists locally. Aborting."; exit 1; fi
	@if git ls-remote --exit-code --heads origin $(BRANCH_NAME_DEPS); then echo "Branch $(BRANCH_NAME_DEPS) already exists remotely. Aborting."; exit 1; fi
	git switch --create $(BRANCH_NAME_DEPS)
	cd client && npm update
	go get -u -t ./...
	go mod tidy
	go mod vendor
	git add .
	@printf "Create commit for dependency updates? [y/N] "; \
	read answer; \
	case "$$answer" in \
		y|Y|yes|YES) git commit -m "build(deps): Update Go and frontend dependencies" ;; \
		*) echo "Skipping commit." ;; \
	esac

changelog:
	git cliff -o CHANGELOG.md
	prettier --write CHANGELOG.md
	@printf "Create commit for changelog update? [y/N] "; \
	read answer; \
	case "$$answer" in \
		y|Y|yes|YES) git commit CHANGELOG.md -m "Update changelog" -m "changelog: ignore" ;; \
		*) echo "Skipping commit." ;; \
	esac

---
title: Development
---

## Development

### Dev Container

#### Usage

This project contains a
[pre-built development container](https://containers.dev/guide/prebuild)
`ghcr.io/jovandeginste/workout-tracker-dev-container:latest` (see
`.devcontainer/devcontainer.json`). Inside the container, just run
`make build-server serve`. After building the project when running it, the app
port will be exposed automatically and you can access the app locally on your
machine via http://localhost:8080/.

#### Build

For building the dev container locally, the variables `GO_VERSION` and
`NODE_VERSION` must be set. Afterwards, the dev container can be built using the
[Dev Container CLI](https://github.com/devcontainers/cli/) via
`devcontainer build --workspace-folder ./.devcontainer-template/ --image-name ghcr.io/jovandeginste/workout-tracker-dev-container`. Here's a full example using PowerShell:

```powershell
$env:GO_VERSION="1.24.1"
$env:NODE_VERSION="22"
devcontainer build --workspace-folder ./.devcontainer-template/ --image-name ghcr.io/jovandeginste/workout-tracker-dev-container
```

### Build and run it yourself

- install go
- clone the repository

```bash
go build ./
./workout-tracker
```

This does not require npm or Tailwind, since the compiled css is included in the
repository.

### Do some development

You need to install Golang and npm.

Because I keep forgetting how to build every component, I created a Makefile.

```bash
# Make everything. This is also the default target.
make all # Run tests and build all components

# Install system dependencies
make install-dev-deps
# Install Javascript libraries
make install-deps

# Testing
make test # Runs all the tests
make test-assets test-go # Run tests for the individual components

# Building
make build # Builds all components
make build-frontend # Builds the frontend assets
make build-templates # Builds the templ templates
make build-server # Builds the web server (includes build-templates)
make build-docker # Performs all builds inside Docker containers, creates a Docker image
make build-swagger # Generates swagger docs



# Running it
make serve # Runs the compiled binary

# Cleanin' up
make clean # Removes build artifacts

# Development
make dev-docker # Runs the server in a docker compose setup
make dev-docker-sqlite # Runs the server in a docker compose setup with SQLite
make dev-docker-clean # Removes volumes created by the dev-docker targets
```

## What is this, technically?

A single binary that runs on any platform, with no dependencies.

The binary contains all assets to serve a web interface, through which you can
upload your GPX files, visualize your tracks and see their statistics and
graphs. The web application is multi-user, with a simple registration and
authentication form, session cookies and JWT tokens). New accounts are inactive
by default. An admin user can activate (or edit, delete) accounts. The default
database storage is a single SQLite file.

## What technologies are used

- Go, with some notable libraries
  - [gpxgo](github.com/tkrajina/gpxgo)
  - [Echo](https://echo.labstack.com/)
  - [Gorm](https://gorm.io)
  - [Spreak](https://github.com/vorlif/spreak)
  - [templ](https://templ.guide/)
  - [HTMX](https://htmx.org/)
- HTML, CSS and JS
  - [Tailwind CSS](https://tailwindcss.com/)
  - [Iconify Design](https://iconify.design/)
  - [FullCalendar](https://fullcalendar.io/)
  - [Leaflet](https://leafletjs.com/)
  - [apexcharts](https://apexcharts.com/)
- Docker

The application uses OpenStreetMap and Esri as its map providers and for
geocoding a GPS coordinate to a location.

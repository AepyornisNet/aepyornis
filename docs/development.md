---
title: Development
---

## Development

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

# Install Javascript libraries
make install-deps

# Testing
make test # Runs all the tests
make test-assets test-go # Run tests for the individual components

# Building
make build # Builds all components
make build-client # Builds the frontend assets
make build-server # Builds the web server
make build-image # Builds the production Docker image using docker/Dockerfile.prod
make swagger # Generates swagger docs



# Running it
make serve # Runs the compiled binary

# Cleanin' up
make clean # Removes build artifacts

# Development
make dev # Runs the server in a docker compose setup with Postgres
make dev-activitypub # Runs two isolated instances for ActivityPub federation testing
make dev-clean # Removes volumes created by development docker compose setups
```

The development setup uses [docker/Dockerfile.dev](../docker/Dockerfile.dev).
Production images use [docker/Dockerfile.prod](../docker/Dockerfile.prod).
Compose files are split by purpose:

- [docker/docker-compose.dev.yaml](../docker/docker-compose.dev.yaml)
- [docker/docker-compose.activitypub.yaml](../docker/docker-compose.activitypub.yaml)
- [docker/docker-compose.yaml](../docker/docker-compose.yaml)

### Test ActivityPub locally with two instances

Add local hostname mappings first:

```bash
echo "127.0.0.1 wt-ap1.test wt-ap2.test" | sudo tee -a /etc/hosts
```

Run:

```bash
make dev-activitypub
```

This starts two independent servers on `https://wt-ap1.test` and
`https://wt-ap2.test`, each with its own Postgres database.

The ActivityPub setup uses a local HTTPS reverse proxy with an internal CA.
Both app containers trust this CA so inter-instance HTTPS requests are accepted.

## What is this, technically?

A single binary that runs on any platform, with no dependencies.

The binary contains all assets to serve a web interface, through which you can
upload your GPX files, visualize your tracks and see their statistics and
graphs. The web application is multi-user, with a simple registration and
authentication form, session cookies and JWT tokens). New accounts are inactive
by default. An admin user can activate (or edit, delete) accounts.

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

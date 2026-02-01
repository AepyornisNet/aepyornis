---
title: Getting started
---

## Getting started

### Docker

Run the latest image from GitHub Container Registry (latest and release images
are available for amd64 and arm64). The current directory is mounted as the data
directory.

```bash
# Latest master build
docker run -p 8080:8080 -v .:/data ghcr.io/jovandeginste/workout-tracker:latest

# Tagged release
docker run -p 8080:8080 -v .:/data ghcr.io/jovandeginste/workout-tracker:2.0.2
docker run -p 8080:8080 -v .:/data ghcr.io/jovandeginste/workout-tracker:2.0
docker run -p 8080:8080 -v .:/data ghcr.io/jovandeginste/workout-tracker:2

# Latest release
docker run -p 8080:8080 -v .:/data ghcr.io/jovandeginste/workout-tracker:release

# Run as non-root user; make sure . is owned by uid 1000
docker run -p 8080:8080 -v .:/data -u 1000:1000 ghcr.io/jovandeginste/workout-tracker
```

Open your browser at `http://localhost:8080`

To persist data and sessions, run:

```bash
docker run -p 8080:8080 \
    -e WT_JWT_ENCRYPTION_KEY=my-secret-key \
    -v $PWD/data:/data \
    ghcr.io/jovandeginste/workout-tracker:master
```

or read the JWT encryption key from a file:

```bash
docker run -p 8080:8080 \
    -e WT_JWT_ENCRYPTION_KEY_FILE=/run/secrets/jwt_encryption_key.txt \
    -v $PWD/jwt_encryption_key.txt:/run/secrets/jwt_encryption_key.txt \
    -v $PWD/data:/data \
    ghcr.io/jovandeginste/workout-tracker:master
```

or use docker compose

```bash
# Create directory that stores your data
mkdir -p /opt/workout-tracker
cd /opt/workout-tracker

# Download the base docker compose file
curl https://raw.githubusercontent.com/jovandeginste/workout-tracker/master/docker/docker-compose.base.yaml --output docker-compose.base.yaml

## For sqlite as database:
curl https://raw.githubusercontent.com/jovandeginste/workout-tracker/master/docker/docker-compose.sqlite.yaml --output docker-compose.yaml

## For postgres as database:
curl https://raw.githubusercontent.com/jovandeginste/workout-tracker/master/docker/docker-compose.postgres.yaml --output docker-compose.yaml
curl https://raw.githubusercontent.com/jovandeginste/workout-tracker/master/docker/postgres.env --output postgres.env

# Start the server
docker compose up -d
```

> **_NOTE:_** If using postgres, configure the parameters in `postgres.env`.

### Natively

Download a
[pre-built binary](https://github.com/jovandeginste/workout-tracker/releases) or
build it yourself (see [Development](#development) below).

Eg. for v2.0.2 on Linux x86_64:

```bash
wget https://github.com/jovandeginste/workout-tracker/releases/download/v2.0.2/workout-tracker-v2.0.2-linux-amd64.tar.gz
tar xf workout-tracker-v2.0.2-linux-amd64.tar.gz
./workout-tracker
```

To persist sessions, run:

```bash
export WT_JWT_ENCRYPTION_KEY=my-secret-key
./workout-tracker
```

or read the JWT encryption key from a file:

```bash
echo "my-secret-key" > ./jwt_encryption_key.txt
export WT_JWT_ENCRYPTION_KEY_FILE=./jwt_encryption_key.txt
./workout-tracker
```

This will create a new database file in the current directory and start the web
server at `http://localhost:8080`.

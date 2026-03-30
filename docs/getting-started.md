---
title: Getting started
---

## Getting started

### Docker Compose

```bash
# Create directory that stores your data
mkdir -p /opt/aepyornis
cd /opt/aepyornis

# Download the base docker compose file
curl https://raw.githubusercontent.com/AepyornisNet/aepyornis/main/docker/docker-compose.yaml --output docker-compose.yaml
curl https://raw.githubusercontent.com/AepyornisNet/aepyornis/main/docker/postgres.env --output postgres.env

# Generate a JWT encryption key
openssl rand -base64 32 > ./jwt_encryption_key.txt

# Start the server
docker compose up -d
```

> **_NOTE:_** Configure the parameters in `postgres.env` before starting.

Open your browser at `http://localhost:8080`

For development and ActivityPub integration testing, use the dedicated compose
files in this repository:

- `docker/docker-compose.dev.yaml`
- `docker/docker-compose.activitypub.yaml`

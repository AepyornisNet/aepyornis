---
title: Configuration
---

## Configuration

The web server looks for a file `workout-tracker.yaml` (or `json` or `toml`) in
the current directory, or takes its configuration from environment variables.
The most important variable is the JWT encryption key. If you don't provide it,
the key is randomly generated every time the server starts, invalidating all
current sessions.

Generate a secure key and write it to `workout-tracker.yaml`:

```bash
echo "jwt_encryption_key_file: ./jwt_encryption_key.txt" > ./workout-tracker.yaml
openssl rand -base64 32 > ./jwt_encryption_key.txt
```

or export it as an environment variable:

```bash
export WT_JWT_ENCRYPTION_KEY="$(openssl rand -base64 32)"
```

See `workout-tracker.example.yaml` for more options and details.

Other environment variables, with their default values:

```bash
WT_BIND="[::]:8080"
WT_WEB_ROOT=""
WT_LOGGING="true"
WT_DEBUG="false"
WT_DATABASE_DRIVER="sqlite"
WT_DSN="./database.db"
WT_REGISTRATION_DISABLED="false"
WT_SOCIALS_DISABLED="false"
WT_DEV="false"
WT_WORKER_DELAY_SECONDS=60
WT_AUTO_IMPORT_ENABLED="false"
WT_OFFLINE="false"
WT_ACTIVITY_PUB_ACTIVE="false"
```

> [!NOTE]
> When using the provided `docker-compose.yaml`, the database driver defaults to
> `postgres`. The environment variables in `postgres.env` configure the
> connection string.

> [!NOTE]  
> Setting `WT_OFFLINE` to `true` runs the app without making external geocoding
> requests (useful for offline environments or to avoid rate limits). In this
> mode, geocoding functions return nil results.

After starting the server, you can access it at <http://localhost:8080> (the
default port). A login form is shown.

If no users are in the database (eg. when starting with an empty database), a
default `admin` user is created with password `admin`. You should change this
password in a production environment.

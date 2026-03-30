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

Aepyornis requires a **PostgreSQL** database. Set the connection string via
`WT_DSN` or in the config file.

Other environment variables, with their default values:

```bash
WT_BIND="[::]:8080"
WT_WEB_ROOT=""
WT_LOGGING="true"
WT_DEBUG="false"
WT_DATABASE_DRIVER="postgres"
WT_DSN="host=localhost user=aepyornis password=aepyornis dbname=aepyornis port=5432 sslmode=disable TimeZone=UTC"
WT_REGISTRATION_DISABLED="false"
WT_SOCIALS_DISABLED="false"
WT_WORKER_DELAY_SECONDS=60
WT_AUTO_IMPORT_ENABLED="false"
WT_OFFLINE="false"
WT_ACTIVITY_PUB_ACTIVE="false"
```

### Hammerhead integration

To enable automatic activity import from a
[Hammerhead Karoo](https://www.hammerhead.io/) device, register an OAuth
application with Hammerhead and set the following variables:

```bash
WT_HAMMERHEAD_CLIENT_ID="your-client-id"
WT_HAMMERHEAD_CLIENT_SECRET="your-client-secret"
WT_HAMMERHEAD_REDIRECT_URI="https://your-instance/profile/apps/hammerhead/callback"
WT_HAMMERHEAD_WEBHOOK_SECRET="your-webhook-secret"
```

| Variable                      | Config key                  | Description                                                   |
| ----------------------------- | --------------------------- | ------------------------------------------------------------- |
| `WT_HAMMERHEAD_CLIENT_ID`     | `hammerhead_client_id`      | OAuth client ID issued by Hammerhead                          |
| `WT_HAMMERHEAD_CLIENT_SECRET` | `hammerhead_client_secret`  | OAuth client secret issued by Hammerhead                      |
| `WT_HAMMERHEAD_REDIRECT_URI`  | `hammerhead_redirect_uri`   | OAuth redirect URI registered with Hammerhead (callback URL)  |
| `WT_HAMMERHEAD_WEBHOOK_SECRET`| `hammerhead_webhook_secret` | Secret used to verify incoming webhook payloads from Karoo    |

When all four variables are set, users can connect their Karoo device under
**Profile → Apps → Hammerhead**.

> [!NOTE]
> The environment variables in `postgres.env` used by `docker-compose.yaml`
> configure the database connection. Edit them before starting the server.

> [!NOTE]  
> Setting `WT_OFFLINE` to `true` runs the app without making external geocoding
> requests (useful for offline environments or to avoid rate limits). In this
> mode, geocoding functions return nil results.

After starting the server, you can access it at <http://localhost:8080> (the
default port). A login form is shown.

If no users are in the database (eg. when starting with an empty database), a
default `admin` user is created with password `admin`. You should change this
password in a production environment.

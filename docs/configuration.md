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

### WebPush & Email Notifications

To enable browser Web Push notifications or email notifications for user activities (likes, comments, follow requests), configure VAPID or email settings:

#### WebPush Notifications

WebPush requires a valid VAPID key pair (ECDSA P-256). You can generate VAPID keys using standard tools (such as `npx web-push generate-vapid-keys`).

```bash
WT_VAPID_PUBLIC_KEY="your-vapid-public-key"
WT_VAPID_PRIVATE_KEY="your-vapid-private-key"
```

| Variable               | Config key          | Description                                                                           |
| ---------------------- | ------------------- | ------------------------------------------------------------------------------------- |
| `WT_VAPID_PUBLIC_KEY`  | `vapid_public_key`  | Public VAPID key provided to browser clients for PushSubscription registration         |
| `WT_VAPID_PRIVATE_KEY` | `vapid_private_key` | Private VAPID key used by the server to sign Web Push messages                        |

When both VAPID keys are provided, the `webpush` provider becomes active. Users can enable WebPush in **Profile → Notifications**, triggering the browser prompt to subscribe to push notifications.

#### Email Notifications

For email notifications:

```bash
WT_MAIL_SENDER_NAME="WorkoutTracker"
WT_MAIL_SENDER_ADDRESS="notifications@example.com"
WT_SMTP_HOST="smtp.example.com"

# Mailjet credentials (optional alternative to SMTP)
WT_MAILJET_PUBLIC_KEY="your-mailjet-public-key"
WT_MAILJET_PRIVATE_KEY="your-mailjet-private-key"
```

| Variable                 | Config key             | Description                                                   |
| ------------------------ | ---------------------- | ------------------------------------------------------------- |
| `WT_MAIL_SENDER_NAME`    | `mail_sender_name`     | Display sender name for outgoing notification emails          |
| `WT_MAIL_SENDER_ADDRESS` | `mail_sender_address`  | Sender email address for outgoing notification emails         |
| `WT_SMTP_HOST`           | `smtp_host`            | Hostname of SMTP server for email delivery                    |
| `WT_MAILJET_PUBLIC_KEY`  | `mailjet_public_key`   | Mailjet API public key                                        |
| `WT_MAILJET_PRIVATE_KEY` | `mailjet_private_key`  | Mailjet API private key                                       |

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

---
title: API
---

## API usage

### API v2 (Recommended)

The new API v2 provides improved pagination, consistent snake_case JSON fields, and better error handling. 

**Base URL:** `/api/v2`

Key features:
- Pagination support for all list endpoints
- Consistent snake_case JSON field naming
- Well-defined response structures
- Type-safe response models

See [API v2 documentation](pkg/api/README.md) for detailed endpoint documentation and examples.

### API v1 (Legacy)

The original API v1 is still available and fully functional at `/api/v1`.

The API is documented using
[swagger](https://editor.swagger.io/?url=https://raw.githubusercontent.com/jovandeginste/workout-tracker/master/docs/swagger.yaml).

### Authentication

You must enable API access for your user, and copy the API key. You can use the
API key as a query parameter (`?api-key=${API_KEY}`) or as a header
(`Authorization: Bearer ${API_KEY}`).

You can configure some tools to automatically upload files to Workout Tracker,
using the `POST /api/v1/import/$program` API endpoint.

### Daily measurements

You can set or update a daily measurement record:

```bash
curl -sSL -H "Authorization: bearer your-api-key" \
  http://localhost:8080/api/v1/daily \
  --data @- <<EOF
{
  "date": "2025-01-13",
  "weight": 70,
  "weight_unit": "kg",
  "height": 178,
  "height_unit": "cm"
}
EOF
```

### Workouts

#### Manual creation

You can create a workout manually:

```bash
curl -sSL -H "Authorization: bearer your-api-key" \
  http://localhost:8080/api/v1/workouts \
  --data @- <<EOF
{
  "name": "Workout name",
  "date": "2025-02-03T10:26",
  "duration_hours": 1,
  "duration_minutes": 10,
  "distance": 13,
  "type": "running"
}
EOF
```

#### Generic upload of a file

The generic upload endpoint takes the recording as body. Prepend the path with
`@` to tell `curl` to read the data from a file:

```bash
curl -sSL -H "Authorization: bearer your-api-key" \
  http://localhost:8080/api/v1/import/generic \
  --data @path/to/recorded.gpx
```

#### FitoTrack automatic GPX export

Read
[their documentation](https://codeberg.org/jannis/FitoTrack/wiki/Auto-Export)
before you continue.

The path to POST to is: `/api/v1/import/fitotrack?api-key=${API_KEY}`

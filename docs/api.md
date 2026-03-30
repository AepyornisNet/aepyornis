---
title: API
---

## API usage

### API v2

The API v2 provides pagination, consistent snake_case JSON fields, and well-defined
response structures.

**Base URL:** `/api/v2`

Key features:

- Pagination support for all list endpoints
- Consistent snake_case JSON field naming
- Well-defined response structures
- Type-safe response models

See the [Swagger documentation](https://editor.swagger.io/?url=https://raw.githubusercontent.com/AepyornisNet/aepyornis/main/docs/swagger.json)
for the full list of endpoints and their parameters.

### Authentication

You must enable API access for your user, and copy the API key. You can use the
API key as a query parameter (`?api-key=${API_KEY}`) or as a header
(`Authorization: Bearer ${API_KEY}`).

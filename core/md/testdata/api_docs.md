# User Management API

Base URL: `https://api.example.com/v2`

All requests require an **Authorization** header with a valid Bearer token.
See [Authentication Guide](https://api.example.com/v2/docs/auth) for details.

## Endpoints

### GET /users/{id}

Retrieves a user by their unique identifier.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| id | string | Yes | The unique identifier of the user |

**Response:**

Returns a JSON object containing the user data:

| Field | Type | Description |
|-------|------|-------------|
| id | string | The unique identifier of the user |
| name | string | Display name |
| email | string | Primary email address |
| role | string | One of: admin, user, viewer |
| created_at | string | ISO 8601 timestamp |

### POST /users

Creates a new user account.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| name | string | Yes | Display name |
| email | string | Yes | Primary email address |
| role | string | No | One of: admin, user, viewer. Default: user |

**Response:**

Returns a JSON object containing the created user.

### DELETE /users/{id}

Deletes a user by their unique identifier.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| id | string | Yes | The unique identifier of the user |

**Response:**

Returns a JSON object containing a confirmation message.

> **Note:** This action is irreversible. All user data will be permanently deleted.

## Rate Limits

All endpoints are subject to rate limiting. See [Rate Limits](https://api.example.com/v2/docs/rate-limits) for current limits.

## Errors

All errors return a JSON object containing:

| Field | Type | Description |
|-------|------|-------------|
| code | string | Machine-readable error code |
| message | string | Human-readable error description |
| request_id | string | Unique identifier for the request |

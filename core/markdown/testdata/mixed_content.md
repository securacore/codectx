# Authentication Guide

This document covers the authentication system including JWT tokens,
OAuth flows, and session management.

## JWT Tokens

JSON Web Tokens are used for stateless authentication.

### Token Structure

A JWT consists of three parts:

- **Header.** Contains the signing algorithm and token type.
- **Payload.** Contains claims about the entity.
- **Signature.** Verifies the token hasn't been tampered with.

```json
{
  "alg": "HS256",
  "typ": "JWT"
}
```

The payload contains standard claims:

| Claim | Description | Required |
|-------|-------------|----------|
| iss | Token issuer | Yes |
| sub | Subject (user ID) | Yes |
| exp | Expiration time | Yes |
| iat | Issued at time | No |

### Refresh Flow

> Refresh tokens should be stored securely and rotated on each use.
> Never expose refresh tokens to client-side JavaScript.

The refresh process works as follows:

1. Client sends expired access token
2. Server validates the refresh token
3. Server issues new access token and refresh token
4. Old refresh token is invalidated

## OAuth Integration

For third-party authentication, we support OAuth 2.0 with the
authorization code flow.

See [OAuth 2.0 RFC](https://tools.ietf.org/html/rfc6749) for details.

---

## Session Management

Sessions are managed via `redis` with a 24-hour TTL.

    session.set(user_id, token, ttl=86400)
    session.get(user_id)

This is an indented code block example.

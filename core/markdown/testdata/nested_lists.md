# Configuration

## Environment Variables

- `DATABASE_URL` - Connection string for the primary database
  - Must include credentials
  - Supports PostgreSQL and MySQL
- `REDIS_URL` - Connection string for cache
- `API_KEY` - External service API key
  - Required for production
  - Optional in development (uses mock)
    - Set `MOCK_API=true` to enable mocking

## Build Steps

1. Install dependencies
2. Configure environment
   - Copy `.env.example` to `.env`
   - Update database credentials
3. Run migrations
4. Start the server
   ```bash
   npm run start
   ```
5. Verify health endpoint

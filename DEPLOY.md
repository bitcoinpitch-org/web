# BitcoinPitch.org Deployment Guide

## Quick Deployment

1. **Clone and setup:**
   ```bash
   git clone <this-repo>
   cd bitcoinpitch-deploy
   cp .env.example .env
   # Edit .env with your production values
   ```

2. **Deploy:**
   ```bash
   make deploy
   ```

## Production Environment Variables

Required environment variables (set in `.env`):

### Database
- `DB_HOST` - PostgreSQL host
- `DB_PORT` - PostgreSQL port (default: 5432)
- `DB_USER` - Database user
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name

### Server
- `PORT` - Application port (default: 8090)

### Security
- `CORS_ALLOWED_ORIGINS` - Allowed origins for CORS
- `RATE_LIMIT_MAX` - Rate limit max requests
- `RATE_LIMIT_EXPIRATION` - Rate limit window (seconds)

### Authentication (Optional)
- `TWITTER_API_KEY` - Twitter OAuth key
- `TWITTER_API_SECRET` - Twitter OAuth secret
- `SMTP_HOST` - Email server host
- `SMTP_PORT` - Email server port
- `SMTP_USER` - Email username
- `SMTP_PASSWORD` - Email password

## Commands

- `make build` - Build the application
- `make docker-up` - Start all services
- `make docker-down` - Stop all services
- `make migrate` - Run database migrations
- `make deploy` - Full deployment (build + migrate + start)

## Monitoring

- Health check: `http://localhost:8090/api/health`
- Logs: `docker-compose logs -f`

## Production Notes

- Ensure PostgreSQL is properly configured and backed up
- Set up SSL/TLS certificates for HTTPS
- Configure proper firewall rules
- Set up log rotation
- Monitor disk space and database growth
- Regular security updates

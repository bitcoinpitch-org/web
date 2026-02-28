# AGENTS.md

## Cursor Cloud specific instructions

### Overview

BitcoinPitch.org is a Go (Fiber) web application with PostgreSQL, server-side Jet templates, and HTMX. Single service, no frontend build step.

### Key paths (hardcoded in code)

The server binary uses hardcoded `/app/` paths for templates, migrations, and i18n. A symlink `/app -> /workspace` must exist for local development. The update script handles this.

### Starting services

1. **PostgreSQL**: `sudo pg_ctlcluster 16 main start`
2. **App server**: `cd /workspace && go run cmd/server/main.go` (runs on `:8090`, auto-runs migrations)

### Database setup (first time only)

```bash
sudo -u postgres psql -c "CREATE USER bitcoinpitch WITH PASSWORD 'dev-password-change-in-production';"
sudo -u postgres psql -c "CREATE DATABASE bitcoinpitch OWNER bitcoinpitch;"
sudo -u postgres psql -d bitcoinpitch -c 'CREATE EXTENSION IF NOT EXISTS "uuid-ossp";'
sudo -u postgres psql -d bitcoinpitch -c "GRANT ALL ON SCHEMA public TO bitcoinpitch;"
```

Then run migrations:
```bash
DATABASE_URL="postgres://bitcoinpitch:dev-password-change-in-production@localhost:5432/bitcoinpitch?sslmode=disable" go run cmd/migrate/main.go -up
```

### Gotchas

- **Migration 15 is a duplicate of 13**: If migrations fail on `config_settings already exists`, force version 15 clean: `UPDATE schema_migrations SET version=15, dirty=false;` then re-run migrations.
- **No test files exist** in the codebase; `go test ./...` passes but reports `[no test files]` for every package.
- **Linter**: Run `golangci-lint run ./...` (needs `$(go env GOPATH)/bin` in PATH). Pre-existing lint warnings exist in the codebase.
- **Admin credentials**: Set `ADMIN_SETUP_TOKEN` and `ADMIN_EMAIL` in `.env`; on first startup the server creates an admin user with email=`ADMIN_EMAIL` and password=`ADMIN_SETUP_TOKEN`.
- **`.env` file**: Copy from `.env.example` or use the `scripts/setup-env.sh` helper. For local dev, set `DB_HOST=localhost`.
- See `README.md` and `DEVELOPMENT.md` for standard `make` commands (build, run, lint, test, docker-up/down).

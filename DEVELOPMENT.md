# BitcoinPitch.org Development Guide

This guide will help you get started with developing, running, and maintaining BitcoinPitch.org. It is designed for beginners and covers all the basics.

---

## 1. Setup

- **Clone the repository:**
  ```bash
  git clone https://github.com/yourusername/bitcoinpitch.org.git
  cd bitcoinpitch.org
  ```

- **Copy the example environment file and edit as needed:**
  ```bash
  cp .env.example .env
  # Edit .env with your favorite editor (nano, vim, code, etc.)
  ```

## 2. Project Structure

The project follows a standard Go project layout with some additional directories for web assets and Docker configuration:

### Root Directories
- `cmd/` - Application entry points and executables
  - `server/` - Main web server application
    - `main.go` - Server initialization and startup
    - Sets up Fiber, middleware, routes, and database
    - Configures Jet template engine with custom functions
  - `migrate/` - Database migration tool
    - `main.go` - CLI tool to run database migrations
    - Handles both up and down migrations
  - `admin-token/` - Admin token generation utility
    - `main.go` - Generate admin authentication tokens
    - Used for setting up initial admin access
  - `testdb/` - Database testing utilities
    - `main.go` - Test database setup and teardown
    - Creates test databases with sample data
  - `test-server/` - Development test server
    - `main.go` - Test server with mock data
    - Used for frontend development without real database

- `internal/` - Core application code that implements the business logic and web interface
  - `handlers/` - HTTP request handlers that process incoming requests
    - `handlers.go` - General page handlers (home, category pages)
    - `auth.go` - Authentication and user management
    - `pitch.go` - Pitch CRUD operations
    - `admin.go` - Admin panel handlers
    - `api.go` - REST API endpoints
    - `validation.go` - Input validation helpers
  - `routes/` - URL routing configuration
    - `routes.go` - Defines all application routes and their handlers
    - Sets up middleware for each route group
    - Configures Jet template context
  - `middleware/` - Request processing components
    - `auth.go` - Authentication checks and user context
    - `security.go` - CORS, CSRF, and security headers
    - `i18n.go` - Internationalization middleware
    - `antispam.go` - Anti-spam protection
    - `health.go` - Health check endpoint
  - `database/` - Database operations
    - `db.go` - Connection management
    - `repository.go` - Query methods and transactions
  - `models/` - Data structures and business logic
    - `models.go` - Core model definitions
    - `user.go` - User model and operations
    - `pitch.go` - Pitch model and operations
    - `config.go` - Configuration settings model
    - `antispam.go` - Anti-spam models
  - `templates/` - Server-side Jet templates
    - `layouts/` - Base templates and common layouts (base.jet)
    - `pages/` - Page-specific templates (home.jet, register.jet, etc.)
    - `partials/` - Reusable template components (header.jet, footer.jet, etc.)
  - `auth/` - Authentication logic
    - `password.go` - Password hashing and validation
    - `twitter.go` - Twitter OAuth integration
    - `totp.go` - TOTP 2FA implementation
    - `admin.go` - Admin authentication
    - `errors.go` - Authentication error handling
  - `config/` - Configuration management
    - `service.go` - Configuration service with caching
  - `i18n/` - Internationalization
    - `i18n.go` - Translation management and loading
  - `antispam/` - Anti-spam system
    - `service.go` - Spam detection and prevention
  - `email/` - Email service
    - `email.go` - SMTP email sending
  - `crypto/` - Cryptographic utilities
    - `nostr.go` - Nostr signature verification
    - `signature.go` - Digital signature utilities
  - `validation/` - Input validation
    - `validation.go` - Form and data validation
  - `static/` - Server-side static assets
    - Internal CSS/JS used by templates
    - Server-generated assets

Note: The `internal/` directory is a Go convention that prevents the code inside it from being imported by other Go projects. This is not about privacy or security, but rather about maintaining a clear API boundary for your application. Code in `internal/` can only be imported by code within the same Go module (your project).

- `migrations/` - Database schema version control
  - SQL files for creating and modifying database tables
  - Each migration has an up and down version
  - Numbered sequentially (e.g., 0000001_initial_schema.sql)
  - Used by the migrate tool in `cmd/migrate/`

- `static/` - Public static assets served directly to clients
  - `css/` - Stylesheets
    - `main.css` - Main application styles
    - `home.css` - Home page specific styles
    - `pitch-form.css` - Pitch form styling
  - `js/` - Client-side JavaScript
    - `main.js` - Main application script with HTMX integration
    - `tutorial.js` - Interactive tutorial system
    - `tutorial-i18n.js` - Tutorial internationalization
  - `img/` - Public images and icons
    - `favicon_io/` - Favicon and app icons
    - Application logos and graphics

- `i18n/` - Translation files
  - `en.json` - English translations
  - `cs.json` - Czech translations
  - JSON format with nested structure for UI elements

- `docker/` - Docker configuration and build files
  - `app/Dockerfile` - Main application container definition
  - `nginx/nginx.conf` - Nginx configuration for production
  - `db/init.sql` - PostgreSQL initialization scripts
  - Used by `docker-compose.yaml` for local development

- `scripts/` - Development and deployment utilities
  - `setup-env.sh` - Development environment setup
  - `test_security.sh` - Security testing scripts

- `volumes/` - Persistent data storage for Docker
  - `postgres/` - PostgreSQL database files
  - `app/` - Application data storage
  - `logs/` - Application logs
  - Not committed to git (in .gitignore)
  - Created automatically by Docker Compose

- `vendor/` - Local copy of Go dependencies
  - Created by `go mod vendor`
  - Ensures reproducible builds
  - Contains exact versions of all dependencies
  - Updated via `go mod tidy` and `go mod vendor`

- `.github/` - GitHub-specific configuration
  - `workflows/` - GitHub Actions CI/CD pipelines
  - Issue and PR templates

### Configuration Files
- `.env` - Environment-specific configuration
  - Database credentials
  - Server settings
  - Security parameters
  - Not committed to git (in .gitignore)
- `docker-compose.yaml` - Docker services definition
  - Defines all services (app, db, nginx)
  - Sets up networking between containers
  - Configures volumes and environment
- `Makefile` - Development workflow automation
  - Common development commands
  - Build and test shortcuts
  - Docker Compose wrappers
- `go.mod` and `go.sum` - Go module management
  - Lists all project dependencies
  - Specifies Go version requirements
  - Ensures dependency integrity

### Documentation
- `README.md` - Project overview and quick start
  - Project description
  - Quick setup instructions
  - Basic usage examples
- `DEVELOPMENT.md` - This development guide
  - Detailed setup instructions
  - Development workflow
  - Project structure
- `IMPLEMENTATION_PLAN.md` - Development roadmap
  - Phase-by-phase implementation plan
  - Feature checklist
  - Progress tracking
- `LICENSE` - Project license (MIT)
  - Terms of use
  - Copyright information

### Development Files
- `.gitignore` - Git version control exclusions
  - Build artifacts
  - Environment files
  - IDE settings
  - Runtime data
- `.cursor/` - Cursor IDE configuration
  - Editor settings
  - Project-specific rules
  - Code snippets
- `drafts/` - Development work in progress
  - Unfinished features
  - Design documents
  - Test content
  - PoC reference implementation

---

## 3. Using Makefile Commands

The project provides a `Makefile` with common commands. You can run these with `make <command>`:

- `make build`         – Build the Go application
- `make run`           – Run the Go application
- `make test`          – Run all Go tests
- `make lint`          – Run the linter
- `make clean`         – Remove build artifacts
- `make docker-up`     – Start all Docker services (app, db, nginx)
- `make docker-down`   – Stop all Docker services
- `make docker-build`  – Rebuild Docker images
- `make db-migrate`    – Run database migrations
- `make db-rollback`   – Rollback the last migration
- `make dev-setup`     – Set up the development environment (modules, .env)

---

## 4. Running the App with Docker

- **Start everything:**
  ```bash
  make docker-up
  ```
- **Stop everything:**
  ```bash
  make docker-down
  ```
- **Rebuild Docker images:**
  ```bash
  make docker-build
  ```

---

## 5. Running the App Locally (without Docker)

- **Build the app:**
  ```bash
  make build
  ```
- **Run the app:**
  ```bash
  make run
  ```

---

## 6. Database Migrations

- **Run migrations:**
  ```bash
  make db-migrate
  ```
- **Rollback last migration:**
  ```bash
  make db-rollback
  ```

---

## 7. Accessing the App

- The app runs at: [http://localhost:8090](http://localhost:8090)
- Health check endpoint: [http://localhost:8090/api/health](http://localhost:8090/api/health)

---

## 8. Testing and Linting

- **Run all tests:**
  ```bash
  make test
  ```
- **Run the linter:**
  ```bash
  make lint
  ```

---

## 9. Troubleshooting

- **Permission errors on volumes:**
  If you see errors about `permission denied` on `volumes/postgres`, run:
  ```bash
  sudo chown -R $(whoami):$(whoami) volumes/postgres
  ```
- **Port conflicts:**
  Make sure nothing else is running on port 8090.
- **Docker not starting:**
  Ensure Docker is running and you have permission to use it.
- **Linter errors about missing dependencies:**
  Run:
  ```bash
  go mod tidy
  go mod vendor
  make lint
  ```

---

## 10. Cleaning Up

- **Remove build artifacts:**
  ```bash
  make clean
  ```
- **Stop and remove all Docker containers, networks, and volumes:**
  ```bash
  make docker-down
  ```

---

## 11. More Information

- See `README.md` for a project overview and structure.
- See `IMPLEMENTATION_PLAN.md` for the step-by-step implementation plan.
- See comments in the `Makefile` for more details on each command.

---

## 12. Environment Variables

The application uses the following environment variables (set in `.env`):

### Database Configuration
- `DB_HOST` - Database host (default: localhost)
- `DB_PORT` - Database port (default: 5432)
- `DB_USER` - Database user
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name

### Server Configuration
- `PORT` - Server port (default: 8090)

### Security Configuration
- `CORS_ALLOWED_ORIGINS` - Comma-separated list of allowed origins for CORS
  - Default: http://localhost:80,http://localhost:8090
  - Production: https://bitcoinpitch.org
- `RATE_LIMIT_MAX` - Maximum number of requests per time window (default: 100)
- `RATE_LIMIT_EXPIRATION` - Rate limit window in seconds (default: 60)

### Authentication Configuration
- `TWITTER_API_KEY` - Twitter OAuth API key
- `TWITTER_API_SECRET` - Twitter OAuth API secret
- `SMTP_HOST` - Email SMTP server host
- `SMTP_PORT` - Email SMTP server port
- `SMTP_USER` - Email SMTP username
- `SMTP_PASSWORD` - Email SMTP password

---

Happy hacking! 🎉 
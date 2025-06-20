# BitcoinPitch.org

A platform for collecting and sharing Bitcoin-related pitches. Users can submit, vote, and share different types of pitches across various Bitcoin-related topics.

## Features

- Submit pitches in different formats (One-liner, SMS, Tweet, Elevator)
- Categorize pitches (Bitcoin, Lightning, Cashu)
- Vote on pitches with real-time scoring
- Share pitches on social media (Twitter, Nostr, Facebook)
- Multiple authentication methods (Trezor, Nostr, Twitter, Email/Password)
- User profiles with privacy settings
- Tag system with filtering capabilities
- Internationalization support (English, Czech)
- Admin panel with configurable settings
- Anti-spam protection
- 2FA support for email/password users
- Mobile-responsive design

## Tech Stack

- Backend: Go with Fiber framework
- Database: PostgreSQL with migrations
- Frontend: HTMX + Vanilla JavaScript
- Templating: Jet template engine (CloudyKit/jet)
- Containerization: Docker and Docker Compose
- Authentication: Multiple methods (hardware wallets, Nostr, OAuth)

## Development Setup

### Prerequisites

- Go 1.21 or later
- Docker and Docker Compose
- Make (optional, for using Makefile commands)

### Quick Start

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/bitcoinpitch.org.git
   cd bitcoinpitch.org
   ```

2. Copy environment file and configure:
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. Start development environment:
   ```bash
   docker-compose up -d
   ```

4. Run database migrations:
   ```bash
   go run cmd/migrate/main.go
   ```

5. Start the development server:
   ```bash
   go run cmd/server/main.go
   ```

The application will be available at http://localhost:8090

## Project Structure

```
.
├── cmd/                    # Application entry points
│   ├── server/            # Main server application
│   ├── migrate/           # Database migration tool
│   ├── admin-token/       # Admin token generation
│   ├── test-server/       # Development test server
│   └── testdb/            # Database testing utilities
├── internal/              # Private application code
│   ├── handlers/          # HTTP request handlers
│   ├── routes/            # URL routing configuration
│   ├── middleware/        # HTTP middleware
│   ├── database/          # Database operations
│   ├── models/            # Data structures and business logic
│   ├── templates/         # Jet HTML templates
│   ├── auth/              # Authentication logic
│   ├── config/            # Configuration management
│   ├── i18n/              # Internationalization
│   ├── antispam/          # Anti-spam system
│   ├── email/             # Email service
│   ├── crypto/            # Cryptographic utilities
│   ├── validation/        # Input validation
│   └── static/            # Server-side static assets
├── static/               # Public static assets
│   ├── css/             # Stylesheets
│   ├── js/              # JavaScript files
│   └── img/             # Images and icons
├── migrations/           # Database schema migrations
├── i18n/                # Translation files
├── docker/              # Docker configuration
├── scripts/             # Development utilities
└── volumes/             # Docker persistent volumes
```

## Development Guidelines

- Follow Go project layout standards
- Use Jet templating for all server-side rendering
- Use semantic HTML and BEM methodology for CSS
- Leverage HTMX for dynamic interactions
- Write tests for all new features
- Follow conventional commits
- Keep documentation up to date

## License

MIT License - see LICENSE file for details 
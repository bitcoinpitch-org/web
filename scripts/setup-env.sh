#!/bin/bash

# Create .env file from template
cat > .env << EOL
# Application
APP_ENV=development
APP_PORT=8080
APP_SECRET=dev-secret-key-change-in-production

# Database
DB_HOST=db
DB_PORT=5432
DB_NAME=bitcoinpitch
DB_USER=bitcoinpitch
DB_PASSWORD=dev-password-change-in-production

# Nginx
NGINX_PORT=80

# Authentication
TREZOR_MESSAGE_PREFIX=BitcoinPitch
NOSTR_RELAY_URL=wss://relay.damus.io
TWITTER_API_KEY=dev-key
TWITTER_API_SECRET=dev-secret
TWITTER_REDIRECT_URL=http://localhost:8090/auth/callback/twitter

# Logging
LOG_LEVEL=debug
LOG_FORMAT=text

# Rate Limiting
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_DURATION=1m
EOL

echo "Created .env file with development settings."
echo "Please review and modify the values as needed."
echo "Remember to change sensitive values in production!" 
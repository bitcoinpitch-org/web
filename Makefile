# BitcoinPitch.org Production Makefile

.PHONY: build run docker-up docker-down docker-build migrate clean

# Build the application
build:
	go build -o bitcoinpitch cmd/server/main.go

# Run the application
run:
	./bitcoinpitch

# Start all services with Docker Compose
docker-up:
	docker-compose up -d

# Stop all services
docker-down:
	docker-compose down

# Build Docker images
docker-build:
	docker-compose build --no-cache

# Run database migrations
migrate:
	go run cmd/migrate/main.go

# Clean build artifacts
clean:
	rm -f bitcoinpitch
	docker-compose down --rmi all --volumes --remove-orphans

# Production deployment
deploy: docker-build migrate docker-up
	@echo "Deployment complete. Check logs with: docker-compose logs -f"

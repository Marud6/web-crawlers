# Web Crawler Orchestrator

A distributed web crawling system with a browser-based management dashboard. All services are containerized and run with a single command.

## Quick Start

```bash
docker compose up --build
```

Open http://localhost to access the dashboard.

## Architecture

| Component | Technology | Description |
|-----------|------------|-------------|
| Orchestrator | Go | REST API that manages crawler worker containers via the Docker socket |
| Crawler | Go | Consumes URLs from RabbitMQ, extracts links, and enqueues discovered URLs |
| Frontend | Vue 3 + Nginx | Dashboard for starting/stopping workers and submitting seed URLs |
| Message Queue | RabbitMQ | Distributes URLs across crawler instances |
| Cache | KeyDB | Tracks visited URLs to prevent duplicate crawling |

The orchestrator dynamically creates and destroys crawler containers at runtime. On shutdown, it stops all spawned crawler containers before exiting.

## Project Structure

```
.
├── crawler/                # Go crawler worker with Dockerfile
├── orchestrator/
│   └── docker-api/         # Go orchestrator API with Dockerfile
├── crawler-GUI/            # Vue 3 frontend application
├── docker-compose.yml      # Multi-service container definition
└── Readme.md
```

## Requirements

- Docker
- Docker Compose

## Features

- Start and stop crawler workers dynamically from the dashboard
- Submit seed URLs to begin crawling
- Deduplicate visited URLs using KeyDB
- Auto-polling dashboard with loading states and error feedback
- Responsive worker card grid with confirmation dialogs
- Graceful shutdown — all crawler containers are cleaned up on `docker compose down`

## Ports

| Service | Port |
|---------|------|
| Frontend | 80 |
| Orchestrator API | 8080 |
| RabbitMQ Management | 15672 |
| RabbitMQ AMQP | 5672 |
| KeyDB | 6379 |

## Screen Shots
<img width="1915" height="869" alt="Snímek obrazovky 2026-03-03 110415" src="https://github.com/user-attachments/assets/5a0e853c-4fb9-4939-8899-c9a847925972" />



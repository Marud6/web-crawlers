# 🕷️ Web Crawler Orchestrator with GUI

Run a scalable, distributed web crawling system **with a browser-based GUI** — all powered by Docker!

---

## 🚀 Quick Start

```bash
docker-compose up --build
Then open your browser at http://localhost:80 to access the management interface.

📦 Project Overview
This project implements a web crawling orchestration platform that allows you to manage distributed crawling tasks with:

🧠 Go-based orchestrator to manage crawling workers

⚙️ Crawling workers in Go, connected via RabbitMQ

📊 Web GUI in Vue.js 

📨 RabbitMQ for message distribution

💾 KeyDB for caching visited URLs

Everything runs in Docker containers with a single docker-compose up.

📁 Project Structure
bash
Zkopírovat
Upravit
.
├── crawler/            # Go crawler service with Dockerfile
├── orchestrator/       # Go orchestrator API
├── web/                # Vue.js frontend app
├── docker-compose.yml  # Multi-service definition
└── README.md           # This file

🧰 Requirements
Docker
Docker Compose


✅ Features
Start/stop crawling workers dynamically

Send new seed URLs to the queue

Avoid duplicate crawling with fast KeyDB cache

Simple browser interface for control and visibility

Logs and status monitoring built in

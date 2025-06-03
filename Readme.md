run docker compose up --build
and on localhost:80 will be gui
this is web crawling workers with management using gui in web browser rabbitMq for distribution and 
keyDB as cashing of links already visited requairment is docker web crawler worker and
orchestrator are written in go
web app is in go 


Web Crawler Orchestrator with GUI
Run a scalable, distributed web crawling system with an easy-to-use web GUI — all powered by Docker!

🚀 Quick Start
bash
Zkopírovat
Upravit
docker-compose up --build
Then open your browser at http://localhost:80 to access the management interface.

What is this?
This project implements a web crawling orchestration platform featuring:

Distributed workers for crawling, coordinated by a central orchestrator

RabbitMQ for reliable task distribution between components

KeyDB for caching already visited links to avoid duplication

A modern web GUI for managing and monitoring crawling workers and tasks

All components are built in Go for high performance and easy deployment.

Components
Orchestrator — manages worker lifecycle, schedules crawling jobs

Workers — perform crawling tasks distributed via RabbitMQ

Web App (GUI) — browser-based dashboard to view status, start/stop workers, and manage seeds

RabbitMQ — message queue for task coordination

KeyDB — caching layer for visited URLs

Requirements
Docker & Docker Compose

Ports 80 (GUI), 5672 (RabbitMQ), and others as configured

Why?
Scalable crawling architecture

Easy management via GUI

Cache layer reduces redundant crawling

Entirely containerized — no manual setup

Feel free to explore, contribute, and scale your own crawlers with ease!
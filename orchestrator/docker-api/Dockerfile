FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o orchestrator main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/orchestrator .
VOLUME /var/run/docker.sock:/var/run/docker.sock
CMD ["./orchestrator"]
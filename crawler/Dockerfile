# crawler/Dockerfile
FROM golang:1.24-alpine

WORKDIR /app

COPY . .

RUN go build crawler.go

CMD ["./crawler"]

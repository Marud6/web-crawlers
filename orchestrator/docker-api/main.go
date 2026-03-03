package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/cors"
)

type containerInfo struct {
	ID    string   `json:"id"`
	Names []string `json:"names"`
	Image string   `json:"image"`
}

type stopRemoveRequest struct {
	ContainerID string `json:"container_id"`
}

type seedRequest struct {
	Seed string `json:"seed"`
}

var (
	dockerCli *client.Client // #20: shared Docker client
	amqpConn  *amqp.Connection
	amqpCh    *amqp.Channel
)

func initDockerClient() {
	var err error
	dockerCli, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	dockerCli.NegotiateAPIVersion(context.Background())
}

func initRabbitMQ() {
	amqpURL := os.Getenv("AMQP_URL")
	if amqpURL == "" {
		amqpURL = "amqp://user:password@rabbitmq:5672/"
	}

	var err error
	amqpConn, err = amqp.Dial(amqpURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	amqpCh, err = amqpConn.Channel()
	if err != nil {
		log.Fatalf("Failed to open AMQP channel: %v", err)
	}

	_, err = amqpCh.QueueDeclare("urls", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}
}

func startContainerHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	resp, err := dockerCli.ContainerCreate(ctx, &container.Config{
		Image: "crawler-image",
		Tty:   false,
	}, &container.HostConfig{
		NetworkMode: "rabbitmqkeydb_default",
	}, &network.NetworkingConfig{}, nil, "")
	if err != nil {
		http.Error(w, "Error creating container: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = dockerCli.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		http.Error(w, "Error starting container: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// #5: return updated container list instead of plain text
	writeContainerList(w)
}

func getRunningContainersHandler(w http.ResponseWriter, r *http.Request) {
	writeContainerList(w)
}

func writeContainerList(w http.ResponseWriter) {
	ctx := context.Background()

	containers, err := dockerCli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		http.Error(w, "Failed to list containers: "+err.Error(), http.StatusInternalServerError)
		return // #2: return after error
	}

	var results []containerInfo
	for _, c := range containers {
		if c.Image == "crawler-image" {
			results = append(results, containerInfo{
				ID:    c.ID[:12],
				Names: c.Names,
				Image: c.Image,
			})
		}
	}

	if results == nil {
		results = []containerInfo{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		log.Printf("Failed to encode container list: %v", err)
	}
}

func stopAndRemoveContainer(containerID string) error {
	ctx := context.Background()

	if err := dockerCli.ContainerStop(ctx, containerID, container.StopOptions{}); err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}

	if err := dockerCli.ContainerRemove(ctx, containerID, container.RemoveOptions{}); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}

	return nil
}

func stopRemoveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req stopRemoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.ContainerID == "" {
		http.Error(w, "container_id is required", http.StatusBadRequest)
		return
	}

	if err := stopAndRemoveContainer(req.ContainerID); err != nil {
		http.Error(w, "Error stopping/removing container: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// #5: return updated container list instead of message object
	writeContainerList(w)
}

func isValidURL(toTest string) bool {
	parsedURL, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return false
	}
	return true
}

func seedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req seedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Seed == "" {
		http.Error(w, "seed is required", http.StatusBadRequest)
		return
	}
	if !isValidURL(req.Seed) {
		http.Error(w, "url not valid", http.StatusBadRequest)
		return
	}

	// #3: proper error responses instead of failOnError/panic
	// #4: use shared amqp connection initialized at startup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := amqpCh.PublishWithContext(ctx,
		"",
		"urls",
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(req.Seed),
		})
	if err != nil {
		http.Error(w, "Failed to publish seed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[x] Sent %s", req.Seed)

	// #5: return updated container list for frontend consistency
	writeContainerList(w)
}

func stopAllCrawlers() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	containers, err := dockerCli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		log.Printf("Failed to list containers for cleanup: %v", err)
		return
	}

	for _, c := range containers {
		if c.Image != "crawler-image" {
			continue
		}
		log.Printf("Stopping crawler container %s...", c.ID[:12])
		if err := dockerCli.ContainerStop(ctx, c.ID, container.StopOptions{}); err != nil {
			log.Printf("Failed to stop %s: %v", c.ID[:12], err)
		}
		if err := dockerCli.ContainerRemove(ctx, c.ID, container.RemoveOptions{}); err != nil {
			log.Printf("Failed to remove %s: %v", c.ID[:12], err)
		}
	}
	log.Println("All crawler containers cleaned up.")
}

func main() {
	initDockerClient()
	initRabbitMQ()
	defer amqpConn.Close()
	defer amqpCh.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/start", startContainerHandler)
	mux.HandleFunc("/containers", getRunningContainersHandler)
	mux.HandleFunc("/stop", stopRemoveHandler)
	mux.HandleFunc("/seed", seedHandler)

	allowedOrigins := os.Getenv("CORS_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "*"
	}

	c := cors.New(cors.Options{
		AllowedOrigins: []string{allowedOrigins},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	handler := c.Handler(mux)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	// #25: graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutting down server...")

		stopAllCrawlers()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Println("Server started at :8080")
	if err := srv.ListenAndServe(); err != http.ErrServerClosed { // #24: check error
		log.Fatalf("ListenAndServe error: %v", err)
	}
	log.Println("Server stopped.")
}

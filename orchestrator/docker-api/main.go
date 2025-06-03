package main

import (
    "context"
    "fmt"
    "net/http"
	"encoding/json"
    "net/url"
    	"log"
    	"time"




            "github.com/rs/cors"
	amqp "github.com/rabbitmq/amqp091-go"

    "github.com/docker/docker/api/types/network"
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/client"
)

type ContainerInfo struct {
	ID    string   `json:"id"`
	Names []string `json:"names"`
	Image string   `json:"image"`
}

type StopRemoveRequest struct {
    ContainerID string `json:"container_id"`
}
type seedRequest struct {
    Seed string `json:"seed"`  // exported field with json tag
}



func startContainerHandler(w http.ResponseWriter, r *http.Request) {

    cli, err := client.NewClientWithOpts(client.FromEnv)
    if err != nil {
        http.Error(w, "Error creating docker client: "+err.Error(), 500)
        return
    }
    cli.NegotiateAPIVersion(context.Background())

    ctx := context.Background()

  resp, err := cli.ContainerCreate(ctx, &container.Config{
      Image: "crawler-image",
      Tty:   false,
  }, &container.HostConfig{
      NetworkMode: "rebitmqkeydb_default",
  }, &network.NetworkingConfig{}, nil, "")
    if err != nil {
        http.Error(w, "Error creating container: "+err.Error(), 500)
        return
    }

   err = cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
   if err != nil {
       http.Error(w, "Error starting container: "+err.Error(), 500)
       return
   }

    fmt.Fprintf(w, "Container started with ID: %s\n", resp.ID)
}


func getRunningContainersHandler(w http.ResponseWriter, r *http.Request){
    ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
	    http.Error(w, "Failed to create Docker client: "+err.Error(), 500)
	}
	cli.NegotiateAPIVersion(ctx)

	imageName := "crawler-image" // change this to your image name

	// Only list running containers
	containers, err := cli.ContainerList(ctx,container.ListOptions{})
	if err != nil {
	       http.Error(w, "Failed to list containers: "+err.Error(), 500)
	}

	var results []ContainerInfo

	for _, container := range containers {
		if container.Image == imageName {
            results = append(results, ContainerInfo{
				ID:    container.ID[:12],
				Names: container.Names,
				Image: container.Image,
			})
		}
	}
jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
	http.Error(w, "Failed to marshal JSON: "+err.Error(), 500)

	}
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
w.Write(jsonData)
}





func stopAndRemoveContainer(containerID string) error {
    ctx := context.Background()

    cli, err := client.NewClientWithOpts(client.FromEnv)
    if err != nil {
        return fmt.Errorf("failed to create docker client: %w", err)
    }
    cli.NegotiateAPIVersion(ctx)

    if err := cli.ContainerStop(ctx, containerID,container.StopOptions{}); err != nil {
        return fmt.Errorf("failed to stop container %s: %w", containerID, err)
    }

    if err := cli.ContainerRemove(ctx, containerID,container.RemoveOptions{}); err != nil {
        return fmt.Errorf("failed to remove container %s: %w", containerID, err)
    }

    return nil
}


func stopRemoveHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req StopRemoveRequest
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

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "message": fmt.Sprintf("Container %s stopped and removed successfully", req.ContainerID),
    })
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

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func seedHandler(w http.ResponseWriter, r *http.Request){
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
    if(!isValidURL(req.Seed)){
     http.Error(w, "url not valid", http.StatusBadRequest)
       return

    }

    conn, err := amqp.Dial("amqp://user:password@rabbitmq:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()
	q, err := ch.QueueDeclare(
		"urls", // name
		true,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = ch.PublishWithContext(ctx,
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(req.Seed),
		})
	failOnError(err, "Failed to publish a message")
	log.Printf(" [x] Sent %s\n", req.Seed)






}


func main() {


    mux := http.NewServeMux()
    mux.HandleFunc("/start", startContainerHandler)
    mux.HandleFunc("/containers", getRunningContainersHandler)
    mux.HandleFunc("/stop", stopRemoveHandler)
    mux.HandleFunc("/seed", seedHandler)


       // Allow all origins:
        c := cors.New(cors.Options{
            AllowedOrigins: []string{"*"},
            AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
            AllowedHeaders: []string{"*"},
        })
    handler := c.Handler(mux)



    fmt.Println("Server started at :8080")
    http.ListenAndServe(":8080", handler)
}

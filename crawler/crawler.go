package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/streadway/amqp"
	"golang.org/x/net/html"
)

var (
	mu         sync.Mutex
	wg         sync.WaitGroup
	ch         *amqp.Channel
	rdb        *redis.Client
	ctx        = context.Background()
	maxWorkers = 10
	sem        chan struct{}
	httpClient = &http.Client{Timeout: 10 * time.Second}
)

func main() {
	amqpURL := os.Getenv("AMQP_URL")
	if amqpURL == "" {
		amqpURL = "amqp://user:password@rabbitmq:5672/"
	}

	conn, err := amqp.Dial(amqpURL)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err = conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare("urls", true, false, false, false, nil)
	failOnError(err, "Failed to declare a queue")

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "keydb:6379"
	}
	rdb = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	sem = make(chan struct{}, maxWorkers)

	urls, err := ch.Consume(
		q.Name,
		"",
		false, // manual ack (#22)
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to register a consumer")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for d := range urls {
			link := string(d.Body)
			log.Printf("Received: %s", link)
			wg.Add(1)
			go crawl(link, d)
		}
	}()

	log.Println("[*] Waiting for messages. To exit press CTRL+C")

	<-sigCh
	log.Println("Shutting down gracefully...")
	wg.Wait()
	log.Println("All workers finished. Exiting.")
}

// isVisited checks Redis for whether a URL has already been crawled.
// On Redis failure, returns false (URL will be re-crawled).
func isVisited(url string) bool {
	exists, err := rdb.SIsMember(ctx, "visited-urls", url).Result()
	if err != nil {
		log.Printf("Redis error: %v", err)
		return false
	}
	return exists
}

func markVisited(url string) {
	err := rdb.SAdd(ctx, "visited-urls", url).Err()
	if err != nil {
		log.Printf("Redis error: %v", err)
	}
}

func crawl(link string, d amqp.Delivery) {
	defer wg.Done() // #1: always called, even on early return

	sem <- struct{}{}        // #16: acquire semaphore slot
	defer func() { <-sem }() // release slot when done

	if isVisited(link) {
		if err := d.Ack(false); err != nil {
			log.Printf("Failed to ack visited URL %s: %v", link, err)
		}
		return
	}

	log.Printf("Crawling: %s", link)
	resp, err := httpClient.Get(link) // #17: client with timeout
	if err != nil {
		log.Printf("GET error for %s: %v", link, err)
		if err := d.Nack(false, true); err != nil {
			log.Printf("Failed to nack %s: %v", link, err)
		}
		return
	}

	links := extractLinks(resp, link)
	resp.Body.Close() // #8: close body right after reading

	markVisited(link)

	mu.Lock() // #23: protect shared channel from concurrent access
	for _, l := range links {
		err := ch.Publish(
			"",
			"urls",
			false,
			false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(l),
			},
		)
		if err != nil {
			log.Printf("Failed to publish link %s: %v", l, err)
		} else {
			log.Printf("Enqueued: %s", l)
		}
	}
	mu.Unlock()

	if err := d.Ack(false); err != nil { // #22: manual ack after success
		log.Printf("Failed to ack %s: %v", link, err)
	}
}

func extractLinks(resp *http.Response, base string) []string {
	tokenizer := html.NewTokenizer(resp.Body)
	var links []string
	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			return links
		case html.StartTagToken, html.SelfClosingTagToken:
			t := tokenizer.Token()
			if t.Data == "a" {
				for _, attr := range t.Attr {
					if attr.Key == "href" {
						href := normalizeURL(attr.Val, base)
						if href != "" {
							links = append(links, href)
						}
					}
				}
			}
		}
	}
}

func normalizeURL(href, base string) string {
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return ""
	}
	resolved := baseURL.ResolveReference(u)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}
	return strings.TrimRight(resolved.String(), "/")
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

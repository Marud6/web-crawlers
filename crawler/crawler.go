package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"context"
	"github.com/streadway/amqp"
	"golang.org/x/net/html"
	"github.com/redis/go-redis/v9"
)

var (
	mu sync.Mutex
	wg sync.WaitGroup
	ch *amqp.Channel
	rdb *redis.Client
    ctx = context.Background()
)

func main() {
    conn, err := amqp.Dial("amqp://user:password@rabbitmq:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()
	ch, err = conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()
	q, err := ch.QueueDeclare("urls", true, false, false, false, nil)
	failOnError(err, "Failed to declare a queue")
	rdb = redis.NewClient(&redis.Options{
		Addr:     "keydb:6379",
		Password: "",
		DB:       0,
	})
	urls, err := ch.Consume(
		q.Name,
		"",
		true,  // auto-ack
		false, // exclusive
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to register a consumer")


	var forever chan struct{}

	go func() {
		for d := range urls {
			link := string(d.Body)
			log.Printf("Received: %s", link)
			wg.Add(1)
			crawl(link)
		}
	}()

	log.Println(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}


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

// limit to urls in list
func crawl(link string) {
	go func() {
	    if(isVisited(link)){
	    return
	    }
		defer wg.Done()
	    fmt.Println("Crawling:", link)
		resp, err := http.Get(link)
		if err != nil {
			fmt.Fprintf(os.Stderr, "GET error: %s\n", err)
			return
		}
        markVisited(link)

		defer resp.Body.Close()

        //data := extractDataFromPage(resp)
		links := extractLinks(resp, link)

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
	}()
}

func extractDataFromPage(resp *http.Response){
return
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




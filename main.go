package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"tsla-mcp/internal/tesla"
)

func main() {
	config, err := tesla.LoadConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	app := tesla.NewApp(config)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{Addr: ":" + port, Handler: app.Handler()}
	if config.RegisterPartner {
		go func() {
			log.Printf("Tesla partner registration enabled for %s", config.PartnerDomain)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			if err := app.RegisterPartnerWithRetry(ctx); err != nil {
				log.Printf("Tesla partner registration failed: %v", err)
				return
			}
			log.Println("Tesla partner account is registered")
		}()
	}

	log.Printf("listening on :%s", port)
	log.Fatal(server.ListenAndServe())
}

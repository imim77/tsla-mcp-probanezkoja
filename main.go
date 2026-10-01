package main

import (
	"log"
	"net/http"

	"tsla-mcp/internal/tesla"
)

func main() {
	config, err := tesla.LoadConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	app := tesla.NewApp(config)
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", app.Handler()))
}

package main

import (
	"fmt"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/jahagaley/phantom/services/github"
	"github.com/jahagaley/phantom/services/pubsub"
)

const (
	health               = "/"
	github_web_hook_path = "/github/webhooks"
	github_setup_path    = "/github/setup"
	pubsub_path          = "/gcp/pubsub"
)

func init() {
	// Setting log level
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// Adding caller information
	log.Logger = log.With().Caller().Logger()
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	// Check the health of the server and return a status code accordingly
	log.Info().Msg("Health check")
	w.WriteHeader(http.StatusOK)
}

func RunServer() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("Error loading .env file: %v", err))
	}

	http.HandleFunc(health, healthHandler)
	http.HandleFunc(github_web_hook_path, github.WebhookHandler)
	http.HandleFunc(github_setup_path, github.SetupHandler)
	http.HandleFunc(pubsub_path, pubsub.Handler)
	http.ListenAndServe(":8080", nil)
}

func main() {

	log.Info().Msg("Running standard Panther server.")
	RunServer()
}

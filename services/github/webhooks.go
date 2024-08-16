package github

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/google/go-github/v50/github"
	"github.com/jahagaley/phantom/services/github/events"
)

// Github Webhook Handler
func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Received a Github Webhook event.")
	payload, err := github.ValidatePayload(r, nil)
	if err != nil {
		log.Error().Msg(fmt.Sprintf("Unable to read payload: %v", err))
	}

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Error().Msg(fmt.Sprintf("Unable to parse event: %v", err))
	}

	switch event := event.(type) {
	case *github.PushEvent:
		events.PushEventHandler(event)
	case *github.CheckRunEvent:
		events.CheckRunEventHandler(event)
	case *github.InstallationRepositoriesEvent:
		events.InstallationRepositoriesEventHandler(r.Context(), event)
	}
}

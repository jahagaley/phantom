package pubsub

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/jahagaley/phantom/checks"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	// Parse the Pub/Sub message from the request body.
	var msg PubSubMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		log.Error().Err(err).Msg("Could not decode body")
		w.WriteHeader(http.StatusBadRequest)
	}

	log.Info().Msg(fmt.Sprintf("Got message with ID: '%s'", msg.Message.ID))
	checkParam, err := DecodeCheckParams(msg.Message.Data)
	if err != nil {
		log.Error().Err(err).Msg("Could not decode CheckParams")
		w.WriteHeader(http.StatusBadRequest)
	}

	go func(checkParam *checks.CheckParams) {
		// TODO - try reprocessing this.
		err := RunIntendedCheck(checkParam)
		log.Error().Err(err).Msg("Failed to complete the processing of check param.")
	}(checkParam)

	// Return a success response.
	w.WriteHeader(http.StatusOK)
}

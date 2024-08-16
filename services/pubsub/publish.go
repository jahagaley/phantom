package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"

	"cloud.google.com/go/pubsub"
	"github.com/jahagaley/phantom/checks"
	"github.com/rs/zerolog/log"
)

// TODO - Make these dynamic
var PUBSUB_PROJECT_ID = "jahagaley"

var PUBSUB_TOPIC_ID = "github-checks-production"

type PubSubMessage struct {
	Message struct {
		Data []byte `json:"data"`
		ID   string `json:"messageId"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

func PublishGithubCheck(checkParams *checks.CheckParams) error {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, PUBSUB_PROJECT_ID)
	if err != nil {
		log.Error().Err(err).Send()
		return err
	}
	defer client.Close()

	t := client.Topic(PUBSUB_TOPIC_ID)
	data, err := EncodeCheckParams(checkParams)
	if err != nil {
		return err
	}
	result := t.Publish(ctx, &pubsub.Message{
		Data: data,
	})
	// Block until the result is returned and a server-generated
	// ID is returned for the published message.
	id, err := result.Get(ctx)
	if err != nil {
		return fmt.Errorf("pubsub: result.Get: %w", err)
	}
	log.Info().Msg(fmt.Sprintf("Published a message; msg ID: %v\n", id))
	return nil
}

func EncodeCheckParams(checkParams *checks.CheckParams) ([]byte, error) {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(checkParams)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeCheckParams(data []byte) (*checks.CheckParams, error) {
	checkParams := checks.CheckParams{}
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&checkParams)
	if err != nil {
		return nil, err
	}
	return &checkParams, nil
}

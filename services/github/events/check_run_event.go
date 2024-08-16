package events

import (
	"fmt"

	"github.com/google/go-github/v50/github"
	checks "github.com/jahagaley/phantom/checks"
	"github.com/jahagaley/phantom/services/pubsub"
	"github.com/rs/zerolog/log"
)

func CheckRunEventHandler(event *github.CheckRunEvent) {

	if event == nil {
		log.Warn().Msg("CheckRunEvent is empty.")
		return
	}

	// If the event is not a rerequest, ignore it.
	if event.GetAction() != "rerequested" {
		log.Info().Msg(fmt.Sprintf("CheckRunEvent was not a rerequest, ignoring: %d", event.GetCheckRun().GetID()))
		return
	}

	checkType, options, err := checks.GetCheckTypeAndOptions(*event.CheckRun.Name)
	if err != nil {
		log.Info().Err(err).Msg("Unable to parse check run rerequested.")
		return
	}
	check := &checks.CheckParams{
		Type:           checkType,
		Name:           *event.CheckRun.Name,
		Owner:          *event.Repo.Owner.Login,
		Repo:           *event.Repo.Name,
		HeadSHA:        *event.CheckRun.HeadSHA,
		Branch:         *event.CheckRun.CheckSuite.HeadBranch,
		DefaultBranch:  *event.Repo.DefaultBranch,
		Options:        options,
		InstallationID: *event.Installation.ID,
		RepoID:         *event.Repo.ID,
	}

	err = pubsub.PublishGithubCheck(check)
	if err != nil {
		log.Err(err).Msg("Unable to Publish CheckParams to PubSub")
	}
}

package events

import (
	"fmt"

	"github.com/google/go-github/v50/github"
	"github.com/jahagaley/phantom/async"
	checks "github.com/jahagaley/phantom/checks/utils"
	"github.com/rs/zerolog/log"
)

func PushEventHandler(event *github.PushEvent) {

	if event == nil {
		log.Warn().Msg("PushEvent is empty.")
		return
	}

	if event.HeadCommit == nil {
		var eventId int64 = 0
		if event.PushID != nil {
			eventId = *event.PushID
		}
		log.Warn().Msg(fmt.Sprintf("Event is missing head commit and isn't a commit, PushEvent Id: %d", eventId))
		return
	}

	// setup params for checks
	check := &checks.CheckParams{
		Type:           checks.SETUP,
		Name:           checks.SETUP,
		Owner:          *event.Repo.Owner.Login,
		Repo:           *event.Repo.Name,
		HeadSHA:        *event.HeadCommit.ID,
		Branch:         event.GetRef()[len("refs/heads/"):],
		DefaultBranch:  *event.Repo.DefaultBranch,
		InstallationID: *event.Installation.ID,
		RepoID:         *event.Repo.ID,
	}

	err := async.PublishGithubCheck(check)
	if err != nil {
		log.Err(err).Msg("Unable to Publish CheckParams to PubSub")
	}

}

package pubsub

import (
	"github.com/rs/zerolog/log"

	. "github.com/jahagaley/phantom/checks"
	"github.com/jahagaley/phantom/services"
)

func RunIntendedCheck(checkParams *CheckParams) error {
	// Get the Client from Github
	githubClient, err := services.NewGithubClientWithInstallationId(checkParams.InstallationID)
	if err != nil {
		log.Err(err).Msg("Failed to create Github client.")
		return err
	}

	// outputs of checks
	var checks []*CheckParams
	switch checkParams.Type {
	case SETUP:
		checks, err = Setup(githubClient, checkParams)
	case PHANTOM_BUILD_IMAGE:
		checks, err = BuildDockerImage(githubClient, checkParams)
	case PHANTOM_TEST_IMAGE:
		checks, err = TestDockerImage(githubClient, checkParams)
	case PHANTOM_RESOURCE_VALIDATION:
		checks, err = ValidateEnvironmentResources(githubClient, checkParams)
	}

	if err != nil {
		log.Err(err).Msg("Failed to run check.")
		return err
	}

	// publish new checks back to pubsub
	for _, c := range checks {
		err = PublishGithubCheck(c)
		if err != nil {
			log.Err(err).Msg("Failed to publish check.")
			return err
		}
	}

	return nil
}

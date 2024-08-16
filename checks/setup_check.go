package checks

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/google/go-github/v50/github"
	"github.com/jahagaley/phantom/utils"
	"github.com/jahagaley/phantomcli/config"
	"github.com/rs/zerolog/log"
)

func Setup(githubClient *github.Client, checkParams *CheckParams) ([]*CheckParams, error) {

	// Important information
	owner := checkParams.Owner
	repo := checkParams.Repo
	headSHA := checkParams.HeadSHA

	// Creating Github Check
	err := CreateCheckRun(githubClient, checkParams)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, err
	}

	// update Check to in progress
	UpdateCheckRunStatus(
		githubClient,
		checkParams,
		STATUS_IN_PROGRESS,
		"Running setup check.",
	)

	// Check to see if any 'phantom.yaml' files exist
	tempDir, err := utils.HandleGithubRepoDownload(githubClient, owner, repo, headSHA)
	if err != nil {
		UpdateCheckRunCompletion(
			githubClient,
			checkParams,
			CONCLUSION_FAILURE,
			fmt.Sprintf("Failed to download files from your repository: %v", err),
			nil,
		)
		os.RemoveAll(tempDir)
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	// TODO: support mono repos and multiple config files.
	if _, err := os.Stat(path.Join(tempDir, "phantom.yaml")); errors.Is(err, os.ErrNotExist) {
		// Updating Check Run on completion
		UpdateCheckRunCompletion(
			githubClient,
			checkParams,
			CONCLUSION_SUCCESS,
			"No 'phantom.yaml' file detected.",
			nil,
		)
		log.Info().Msg("No 'phantom.yaml' file detected.")
		return nil, err
	}

	cfg, err := config.GetConfigFromPath(tempDir)
	if err != nil {
		// Updating Check Run on completion
		UpdateCheckRunCompletion(
			githubClient,
			checkParams,
			CONCLUSION_FAILURE,
			"Unable to load your 'phantom.yaml' file.",
			nil,
		)
		log.Warn().Msg("Unable to load your 'phantom.yaml' file.")
		return nil, err
	}

	// Get follow up checks
	var output []*CheckParams

	// Check if any builds need to be done.
	for _, b := range cfg.Builds {
		newCheck := &CheckParams{
			Type:           PHANTOM_BUILD_IMAGE,
			Name:           fmt.Sprintf(PHANTOM_BUILD_IMAGE_PATTERN, b.Name),
			Owner:          checkParams.Owner,
			Repo:           checkParams.Repo,
			HeadSHA:        checkParams.HeadSHA,
			Branch:         checkParams.Branch,
			DefaultBranch:  checkParams.DefaultBranch,
			InstallationID: checkParams.InstallationID,
			RepoID:         checkParams.RepoID,
			Options: map[string]string{
				"name": b.Name,
			},
		}

		output = append(output, newCheck)
	}

	// Validate resources for each environment
	for _, e := range cfg.Environments {
		newCheck := &CheckParams{
			Type:           PHANTOM_RESOURCE_VALIDATION,
			Name:           fmt.Sprintf(PHANTOM_RESOURCE_VALIDATION_PATTERN, e.Environment),
			Owner:          checkParams.Owner,
			Repo:           checkParams.Repo,
			HeadSHA:        checkParams.HeadSHA,
			Branch:         checkParams.Branch,
			DefaultBranch:  checkParams.DefaultBranch,
			InstallationID: checkParams.InstallationID,
			RepoID:         checkParams.RepoID,
			Options: map[string]string{
				"name": e.Environment,
			},
		}

		output = append(output, newCheck)
	}

	// Updating Check Run on completion
	UpdateCheckRunCompletion(
		githubClient,
		checkParams,
		CONCLUSION_SUCCESS,
		"Setup complete.",
		nil,
	)

	return output, nil
}

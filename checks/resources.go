package checks

import (
	"fmt"
	"os"

	"github.com/google/go-github/v50/github"
	"github.com/jahagaley/phantom/utils"
	"github.com/jahagaley/phantomcli/cloud/gcp"
	"github.com/jahagaley/phantomcli/config"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rs/zerolog/log"
)

func ValidateEnvironmentResources(githubClient *github.Client, checkParams *CheckParams) ([]*CheckParams, error) {
	// Creating Github Check
	err := CreateCheckRun(githubClient, checkParams)
	if err != nil {
		return nil, err
	}

	return ValidateEnvironmentResourcesWithCheck(githubClient, checkParams)
}

func ValidateEnvironmentResourcesWithCheck(githubClient *github.Client, checkParams *CheckParams) ([]*CheckParams, error) {

	owner, repo, headSHA := checkParams.Owner, checkParams.Repo, checkParams.HeadSHA
	// update Check to in progress
	UpdateCheckRunStatus(
		githubClient,
		checkParams,
		STATUS_IN_PROGRESS,
		"Validating Resources for selected environment.",
	)

	// Download repo contents
	tempDir, err := utils.HandleGithubRepoDownload(githubClient, owner, repo, headSHA)
	if err != nil {
		UpdateCheckRunCompletion(
			githubClient,
			checkParams,
			CONCLUSION_FAILURE,
			"Unable to download the the contents of commit to check resources",
			nil,
		)
		os.RemoveAll(tempDir)
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	// Get the config file from the repo
	cfg, err := config.GetConfigFromPath(tempDir)
	if err != nil {
		// Updating Check Run on completion
		UpdateCheckRunCompletion(
			githubClient,
			checkParams,
			CONCLUSION_FAILURE,
			"Unable to get 'phantom.yaml' file for testing image.",
			nil,
		)
		return nil, err
	}

	// Get the test
	environment, ok := checkParams.Options["name"]
	if !ok {
		return nil, fmt.Errorf("Unable to find environment from check params.")
	}

	env := cfg.GetDeployEnvironment(environment)
	if env == nil {
		err = fmt.Errorf("Unable to get environment with name '%s", environment)

		UpdateCheckRunCompletion(
			githubClient,
			checkParams,
			CONCLUSION_FAILURE,
			err.Error(),
			nil,
		)
		return nil, err
	}
	log.Info().Msg(fmt.Sprintf("Running resource validation for '%s'", environment))

	// Get IAM access token.
	token, err := GetAuthAccessToken()
	if err != nil {
		UpdateCheckRunCompletion(
			githubClient,
			checkParams,
			CONCLUSION_FAILURE,
			"Failed to get access token to connect with your cloud.",
			nil,
		)
		return nil, err
	}

	// TODO - pull this from a DB
	provider := gcp.Provider{
		IAMToken:         token,
		Project:          "jahagaley",
		ArtifactRegistry: "us-docker.pkg.dev/jahagaley/phantom",
	}

	// Get the status of the resources.
	cloudManager := gcp.NewGoogleCloudManager(provider, &cfg)
	statuses, err := cloudManager.GetStatus(env)
	if err != nil {
		UpdateCheckRunCompletion(
			githubClient,
			checkParams,
			CONCLUSION_FAILURE,
			"Failed to get status of cloud resources.",
			nil,
		)
		return nil, err
	}

	// generate output for checks
	t := table.NewWriter()
	t.AppendHeader(table.Row{"Type", "Name", "Exists", "Requires Update"})
	for _, s := range statuses {
		t.AppendRow([]interface{}{s.Type, s.Name, s.Exists, s.RequiresUpdate})
		t.AppendSeparator()
	}
	t.SortBy([]table.SortBy{
		{Name: "Type", Mode: table.Asc},
	})

	outputMarkDown := t.RenderMarkdown()

	// Updating Check Run on completion
	UpdateCheckRunCompletionWithOutput(
		githubClient,
		checkParams,
		CONCLUSION_SUCCESS,
		fmt.Sprintf("Resource validation completed for environment '%s'.", environment),
		outputMarkDown,
		nil,
	)

	return nil, err
}

package checks

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v50/github"

	"connectrpc.com/connect"
	apipb "github.com/jahagaley/phantomapi/phantom/api/v1"
	api "github.com/jahagaley/phantomapi/phantom/api/v1/v1connect"

	"github.com/jahagaley/phantom/utils"
	"github.com/jahagaley/phantomcli/config"
	"github.com/rs/zerolog/log"
)

const (
	BUILD_ID_PATTERN = "build-%d-%s"
)

func BuildDockerImage(githubClient *github.Client, checkParams *CheckParams) ([]*CheckParams, error) {
	// Creating Github Check
	err := CreateCheckRun(githubClient, checkParams)
	if err != nil {
		return nil, err
	}

	return BuildDockerImageWithCheck(githubClient, checkParams)
}

func BuildDockerImageWithCheck(githubClient *github.Client, checkParams *CheckParams) ([]*CheckParams, error) {
	owner, repo, headSHA := checkParams.Owner, checkParams.Repo, checkParams.HeadSHA
	// update Check to in progress
	UpdateCheckRunStatus(
		githubClient,
		checkParams,
		STATUS_IN_PROGRESS,
		"Running Docker build for image.",
	)

	// Download repo contents
	tempDir, err := utils.HandleGithubRepoDownload(githubClient, owner, repo, headSHA)
	if err != nil {
		UpdateCheckRunCompletion(
			githubClient,
			checkParams,
			CONCLUSION_FAILURE,
			"Unable to download the the contents of commit to run build.",
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
			"Unable to get 'phantom.yaml' file for building image.",
			nil,
		)
		return nil, err
	}

	buildName := checkParams.Options["name"]
	build := cfg.GetBuild(buildName)
	if build == nil {
		UpdateCheckRunCompletion(
			githubClient,
			checkParams,
			CONCLUSION_FAILURE,
			"Unable to get the build from the config file.",
			nil,
		)
		return nil, fmt.Errorf("Unable to find build with name '%s'", buildName)
	}

	status, err := executeBuild(githubClient, checkParams, build)
	if err != nil {
		UpdateCheckRunCompletion(
			githubClient,
			checkParams,
			CONCLUSION_FAILURE,
			"Unable to complete build execution.",
			nil,
		)
		return nil, err
	}

	// Updating Check Run on completion
	UpdateCheckRunCompletion(
		githubClient,
		checkParams,
		status,
		fmt.Sprintf("Docker build completed with status '%s'.", status),
		nil,
	)

	var output []*CheckParams
	// Check if any checks are needed for Phantom.
	if len(cfg.Tests) > 0 {
		for _, test := range cfg.Tests {
			newCheck := &CheckParams{
				Type:           PHANTOM_TEST_IMAGE,
				Name:           fmt.Sprintf(PHANTOM_TEST_IMAGE_PATTERN, test.Name),
				Owner:          checkParams.Owner,
				Repo:           checkParams.Repo,
				HeadSHA:        checkParams.HeadSHA,
				Branch:         checkParams.Branch,
				DefaultBranch:  checkParams.DefaultBranch,
				InstallationID: checkParams.InstallationID,
				RepoID:         checkParams.RepoID,
				Options: map[string]string{
					"name": test.Name,
				},
			}

			output = append(output, newCheck)
		}
	}

	return output, nil
}

func executeBuild(githubClient *github.Client, checkParams *CheckParams, build *config.PhantomBuild) (string, error) {
	ctx := context.Background()
	log.Info().Msg("Starting Docker Build Execution")

	// Create ConnectRPC Client
	gitClient := api.NewGitServiceClient(http.DefaultClient, utils.API_URL)

	// Get the installation from the GitService
	getInstallationRequest := &apipb.GetInstallationRequest{
		Name: fmt.Sprintf("projects/-/installations/%d", checkParams.InstallationID),
	}
	installationRes, err := gitClient.GetInstallation(ctx, connect.NewRequest(getInstallationRequest))
	if err != nil {
		log.Err(err).Msg("Error getting installation, missing installation cannot be created")
		return "", err
	}

	// TODO - clean up this code
	// Split the name field of the installation response to get the project name
	project := strings.Split(installationRes.Msg.GetName(), "/")[1]

	// Build Id
	buildId := fmt.Sprintf(BUILD_ID_PATTERN, checkParams.RepoID, build.Name)

	// Get the build from the GitService
	getBuildRequest := &apipb.GetBuildRequest{
		Name: fmt.Sprintf("projects/%s/builds/%s", project, buildId),
	}
	buildRes, err := gitClient.GetBuild(ctx, connect.NewRequest(getBuildRequest))
	if err != nil && connect.CodeOf(err) == connect.CodeNotFound {
		// Create the build if it doesn't exist
		createBuildRequest := &apipb.CreateBuildRequest{
			Parent:  fmt.Sprintf("projects/%s", project),
			BuildId: buildId,
			Build: &apipb.Build{
				DisplayName: build.Name,
				Repository:  fmt.Sprintf("projects/%s/repositories/%d", project, checkParams.RepoID),
				Path:        build.Path,
				File:        build.File,
			},
		}

		buildRes, err = gitClient.CreateBuild(ctx, connect.NewRequest(createBuildRequest))
		if err != nil {
			log.Err(err).Msg("Error creating build")
			return "", err
		}
	} else if err != nil {
		log.Err(err).Msg("Error getting build")
		return "", err
	}

	// Get the download url
	downloadUrl, _, err := githubClient.Repositories.GetArchiveLink(
		context.Background(),
		checkParams.Owner,
		checkParams.Repo,
		github.Tarball,
		&github.RepositoryContentGetOptions{
			Ref: checkParams.HeadSHA,
		},
		true,
	)
	if err != nil {
		log.Err(err).Msg("Error getting download url")
		return "", err
	}

	// Create the execution
	createExecutionRequest := &apipb.CreateExecutionRequest{
		Parent: fmt.Sprintf("projects/%s", project),
		Execution: &apipb.Execution{
			Branch: checkParams.Branch,
			Commit: checkParams.HeadSHA,
			BuildOrTest: &apipb.Execution_BuildRequest_{
				BuildRequest: &apipb.Execution_BuildRequest{
					Build:    buildRes.Msg.Name,
					Download: downloadUrl.String(),
				},
			},
		},
	}

	executionRes, err := gitClient.CreateExecution(ctx, connect.NewRequest(createExecutionRequest))
	if err != nil {
		log.Err(err).Msg("Error creating execution")
		return "", err
	}

	// In a loop, check the state of the build and sleep if it is not finished
	// Only loop for 15 minutes
	// TODO - Try long running operations
	for i := 0; i < 60; i++ {
		// Get the execution
		getExecutionRequest := &apipb.GetExecutionRequest{
			Name: executionRes.Msg.GetName(),
		}
		executionRes, err = gitClient.GetExecution(ctx, connect.NewRequest(getExecutionRequest))
		if err != nil {
			log.Err(err).Msg("Error getting execution")
			return "", err
		}

		// Check the state of the execution
		if executionRes.Msg.GetState() == apipb.Execution_STATE_SUCCESS {
			// Return the status of the execution
			return CONCLUSION_SUCCESS, nil
		} else if executionRes.Msg.GetState() == apipb.Execution_STATE_FAILURE {
			// Return the status of the execution
			return CONCLUSION_FAILURE, nil
		} else if executionRes.Msg.GetState() == apipb.Execution_STATE_CANCELLED {
			// Returns the status of the execution
			return CONCLUSION_CANCELLED, nil
		} else if executionRes.Msg.GetState() == apipb.Execution_STATE_UNSPECIFIED {
			return CONCLUSION_FAILURE, nil
		}

		time.Sleep(15 * time.Second)
	}

	return CONCLUSION_TIMED_OUT, nil
}

package checks

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/go-github/v50/github"
	apipb "github.com/jahagaley/phantomapi/phantom/api/v1"
	api "github.com/jahagaley/phantomapi/phantom/api/v1/v1connect"

	. "github.com/jahagaley/phantom/checks/utils"
	"github.com/jahagaley/phantom/utils"
	"github.com/jahagaley/phantomcli/config"
	"github.com/rs/zerolog/log"
)

const (
	TEST_ID_PATTERN = "test-%d-%s"
)

func TestDockerImage(githubClient *github.Client, checkParams *CheckParams) ([]*CheckParams, error) {
	// Creating Github Check
	err := CreateCheckRun(githubClient, checkParams)
	if err != nil {
		return nil, err
	}

	return TestDockerImageWithCheck(githubClient, checkParams)
}

func TestDockerImageWithCheck(githubClient *github.Client, checkParams *CheckParams) ([]*CheckParams, error) {

	owner, repo, headSHA := checkParams.Owner, checkParams.Repo, checkParams.HeadSHA
	// update Check to in progress
	UpdateCheckRunStatus(
		githubClient,
		checkParams,
		STATUS_IN_PROGRESS,
		"Running Docker tests for images.",
	)

	// Download repo contents
	tempDir, err := utils.HandleGithubRepoDownload(githubClient, owner, repo, headSHA)
	if err != nil {
		UpdateCheckRunCompletion(
			githubClient,
			checkParams,
			CONCLUSION_FAILURE,
			"Unable to download the the contents of commit to run tests.",
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
	testName, ok := checkParams.Options["name"]
	if !ok {
		return nil, fmt.Errorf("Unable to find test name from check params")
	}

	test := cfg.GetTest(testName)
	if test == nil {
		err = fmt.Errorf("Unable to get test with name '%s", testName)
		UpdateCheckRunCompletion(
			githubClient,
			checkParams,
			CONCLUSION_FAILURE,
			err.Error(),
			nil,
		)
		return nil, err
	}
	log.Info().Msg(fmt.Sprintf("Running test for '%s'", testName))

	// Get the build used first
	build := cfg.GetBuild(test.Build)
	if build == nil {
		err = fmt.Errorf("Unable to get build '%s' used by test '%s'", test.Build, test.Name)
		UpdateCheckRunCompletion(
			githubClient,
			checkParams,
			CONCLUSION_FAILURE,
			err.Error(),
			nil,
		)
		return nil, err
	}

	status, err := executeTest(githubClient, checkParams, test)
	if err != nil {
		UpdateCheckRunCompletion(
			githubClient,
			checkParams,
			CONCLUSION_FAILURE,
			"Unable to complete test execution.",
			nil,
		)
		return nil, err
	}

	// Updating Check Run on completion
	UpdateCheckRunCompletion(
		githubClient,
		checkParams,
		status,
		fmt.Sprintf("Test execution completed with status '%s'.", status),
		nil,
	)

	return nil, err
}

func executeTest(githubClient *github.Client, checkParams *CheckParams, test *config.PhantomTest) (string, error) {
	// Create a client
	ctx := context.Background()
	log.Info().Msg("Starting Docker Test Execution")

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

	// Build and Test Id
	buildId := fmt.Sprintf(BUILD_ID_PATTERN, checkParams.RepoID, test.Build)
	testId := fmt.Sprintf(TEST_ID_PATTERN, checkParams.RepoID, test.Name)

	// Get the test from the GitService
	getTestRequest := &apipb.GetTestRequest{
		Name: fmt.Sprintf("projects/%s/tests/%s", project, testId),
	}
	testRes, err := gitClient.GetTest(ctx, connect.NewRequest(getTestRequest))
	if err != nil && connect.CodeOf(err) == connect.CodeNotFound {
		// Create the test if it doesn't exist
		createTestRequest := &apipb.CreateTestRequest{
			Parent: fmt.Sprintf("projects/%s", project),
			TestId: testId,
			Test: &apipb.Test{
				DisplayName: test.Name,
				Repository:  fmt.Sprintf("projects/%s/repositories/%d", project, checkParams.RepoID),
				Build:       fmt.Sprintf("projects/%s/builds/%s", project, buildId),
				Command:     test.Command,
			},
		}

		testRes, err = gitClient.CreateTest(ctx, connect.NewRequest(createTestRequest))
		if err != nil {
			log.Err(err).Msg("Error creating test")
			return "", err
		}
	} else if err != nil {
		log.Err(err).Msg("Error getting test")
		return "", err
	}

	// Create the execution
	createExecutionRequest := &apipb.CreateExecutionRequest{
		Parent: fmt.Sprintf("projects/%s", project),
		Execution: &apipb.Execution{
			Branch: checkParams.Branch,
			Commit: checkParams.HeadSHA,
			BuildOrTest: &apipb.Execution_TestRequest_{
				TestRequest: &apipb.Execution_TestRequest{
					Test: testRes.Msg.Name,
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
	for {
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
			// Sleep for 5 seconds
			return CONCLUSION_CANCELLED, nil
		} else if executionRes.Msg.GetState() == apipb.Execution_STATE_UNSPECIFIED {
			return CONCLUSION_FAILURE, nil
		}

		// Sleep
		time.Sleep(15 * time.Second)
	}
}

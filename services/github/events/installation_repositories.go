package events

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/go-github/v50/github"

	"connectrpc.com/connect"
	apipb "github.com/jahagaley/phantomapi/phantom/api/v1"
	api "github.com/jahagaley/phantomapi/phantom/api/v1/v1connect"

	"github.com/jahagaley/phantom/utils"
)

func InstallationRepositoriesEventHandler(ctx context.Context, event *github.InstallationRepositoriesEvent) {

	if event == nil {
		log.Warn().Msg("InstallationEvent is empty.")
		return
	}

	// Get the installation Id from the event
	installationID := event.GetInstallation().GetID()

	if event.GetAction() == "added" {
		// Create a Phantom Git API client
		gitClient := api.NewGitServiceClient(http.DefaultClient, utils.API_URL)

		// Get the installation from the GitService
		getInstallationRequest := &apipb.GetInstallationRequest{
			Name: fmt.Sprintf("projects/-/installations/%d", installationID),
		}
		res, err := gitClient.GetInstallation(ctx, connect.NewRequest(getInstallationRequest))
		installation := res.Msg
		if status.Code(err) == codes.NotFound {
			log.Warn().Err(err).Msg("Installation not found")
			return
		} else if err != nil {
			log.Err(err).Msg("Error getting installation")
			return
		}

		// TODO - clean up this code
		// Split the name field of the installation response to get the project name
		// using the default string split function. try to clean up later
		project := strings.Split(installation.GetName(), "/")[1]
		// For each repository in the installation, create a new repository
		for _, repo := range event.RepositoriesAdded {
			// Create a new repository
			repository := &apipb.Repository{
				DisplayName:    repo.GetName(),
				Owner:          repo.GetOwner().GetLogin(),
				DefaultBranch:  repo.GetDefaultBranch(),
				Installation:   fmt.Sprintf("projects/%s/installations/%d", project, installationID),
				RepositoryType: apipb.Repository_REPOSITORY_TYPE_GITHUB,
			}

			// Create a new repository in the GitService
			createRepositoryRequest := &apipb.CreateRepositoryRequest{
				Parent:       fmt.Sprintf("projects/%s", project),
				RepositoryId: fmt.Sprintf("%d", repo.GetID()),
				Repository:   repository,
			}
			_, err = gitClient.CreateRepository(ctx, connect.NewRequest(createRepositoryRequest))
			if err != nil {
				log.Err(err).Msgf("Error creating repository: %s", repo.GetName())
			}
		}

	} else if event.GetAction() == "removed" {
		// TODO - support removing repositories
	}
}

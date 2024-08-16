package github

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"connectrpc.com/connect"
	apipb "github.com/jahagaley/phantomapi/phantom/api/v1"
	api "github.com/jahagaley/phantomapi/phantom/api/v1/v1connect"

	"github.com/jahagaley/phantom/services"
	"github.com/jahagaley/phantom/utils"
)

// TODO - Support Github app removals
func SetupHandler(w http.ResponseWriter, r *http.Request) {
	// Get the installation_id and state query parameters from the request
	installationID := r.URL.Query().Get("installation_id")
	state := r.URL.Query().Get("state")

	// Validate that the installation_id and state parameters exist
	if installationID == "" {
		log.Info().Msg("Missing installation_id parameter")
		http.Error(w, "Missing installation_id parameter", http.StatusBadRequest)
		return
	}
	if state == "" {
		log.Info().Msg("Missing state parameter")
		http.Redirect(w, r, utils.SITE_URL, http.StatusMovedPermanently)
		return
	}

	// Convert installationID to int64 and validate that it is a valid integer
	installationIDInt, err := strconv.ParseInt(installationID, 10, 64)
	if err != nil {
		log.Error().Err(err).Msg("Invalid installation_id parameter")
		http.Error(w, "Invalid installation_id parameter", http.StatusBadRequest)
		return
	}

	// Create a GitHub client using the installation ID
	client, err := services.NewGithubClientWithInstallationId(installationIDInt)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create GitHub client")
		http.Error(w, "Failed to create GitHub client", http.StatusInternalServerError)
		return
	}

	// Use the GitHub client to get the repositories from the installation ID
	repos, _, err := client.Apps.ListRepos(r.Context(), nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get repositories")
		http.Error(w, "Failed to get repositories", http.StatusInternalServerError)
		return
	}

	// Create a Phantom Git API client
	gitClient := api.NewGitServiceClient(http.DefaultClient, utils.API_URL)

	// Creates a new installation for the project
	createInstallationRequest := &apipb.CreateInstallationRequest{
		Parent:         fmt.Sprintf("projects/%s", state),
		InstallationId: installationID,
		Installation:   &apipb.Installation{},
	}
	_, err = gitClient.CreateInstallation(r.Context(), connect.NewRequest(createInstallationRequest))

	if status.Code(err) == codes.AlreadyExists {
		log.Warn().Err(err).Msg("Installation already exists")
		return
	} else if err != nil {
		log.Error().Err(err).Msg("Failed to create installation")
		http.Error(w, "Failed to create installation", http.StatusInternalServerError)
		return
	}

	// Iterator over the repositories and print the id, name, owner, and default branch
	for _, repo := range repos.Repositories {
		createRepositoryRequest := &apipb.CreateRepositoryRequest{
			Parent:       fmt.Sprintf("projects/%s", state),
			RepositoryId: strconv.FormatInt(repo.GetID(), 10),
			Repository: &apipb.Repository{
				DisplayName:    repo.GetName(),
				Owner:          repo.GetOwner().GetLogin(),
				DefaultBranch:  repo.GetDefaultBranch(),
				Installation:   fmt.Sprintf("projects/%s/installations/%s", state, installationID),
				RepositoryType: apipb.Repository_REPOSITORY_TYPE_GITHUB,
			},
		}
		repository, err := gitClient.CreateRepository(r.Context(), connect.NewRequest(createRepositoryRequest))
		if err != nil {
			log.Error().Err(err).Msg("Failed to create repository")
			http.Error(w, "Failed to create repository", http.StatusInternalServerError)
			return
		}

		log.Info().Interface("repository", repository).Msg(fmt.Sprintf("Created repository: %s", repo.GetName()))
	}

	log.Info().Msg("Successfully created repositories")
	// Return a success response
	http.Redirect(w, r, utils.SITE_URL, http.StatusMovedPermanently)
}

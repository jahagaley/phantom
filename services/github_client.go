package services

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v50/github"
	"github.com/jahagaley/phantom/utils"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

const (
	GITHUB_APP_ID = "PHANTOM_APP_ID"
)

func createInstallationToken(currentClient *github.Client, installationID int64) (string, error) {
	// Create an installation token for the repository
	token, _, err := currentClient.Apps.CreateInstallationToken(context.Background(), installationID, nil)
	if err != nil {
		log.Warn().Err(err).Msg("Error creating installation token")
		return "", err
	}

	return token.GetToken(), nil
}

func NewGithubClientWithInstallationId(installationID int64) (*github.Client, error) {
	client, err := NewGithubClient()
	if err != nil {
		log.Err(err).Msg("Error creating Github client")
		return nil, err
	}

	// Create a new OAuth2 token source with the installation access token
	token, err := createInstallationToken(client, installationID)
	if err != nil {
		log.Err(err).Msg("Unable to create token for Github installation")
		return nil, err
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	// Create a new OAuth2 HTTP client with the token source
	tc := oauth2.NewClient(context.Background(), ts)

	// Create a new GitHub client with the HTTP client
	client = github.NewClient(tc)

	return client, nil
}

func NewGithubClient() (*github.Client, error) {
	// Getting secret from Secret Manager
	privateKey, err := utils.AccessSecretVersion("projects/jahagaley/secrets/github_pem/versions/latest")
	if err != nil {
		log.Err(err).Msg("Error reading private key - from named secret")
		return nil, err
	}

	// Parse the private key into an rsa.PrivateKey object
	key, err := jwt.ParseRSAPrivateKeyFromPEM(privateKey)
	if err != nil {
		log.Warn().Err(err).Msg("Error parsing private key")
		return nil, err
	}

	// Create a new JWT for the GitHub App
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute * 10)),
		Issuer:    GITHUB_APP_ID,
	})
	tokenString, err := token.SignedString(key)
	if err != nil {
		return nil, err
	}

	// Set up the OAuth2 token for authentication using the JWT
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tokenString})
	tc := oauth2.NewClient(context.Background(), ts)

	// Create a new GitHub client with the HTTP client
	client := github.NewClient(tc)

	return client, nil
}

package checks

import (
	"context"

	"github.com/rs/zerolog/log"
	auth "golang.org/x/oauth2/google"
)

func GetAuthAccessToken() (string, error) {
	scopes := []string{
		"https://www.googleapis.com/auth/cloud-platform",
	}

	credentials, err := auth.FindDefaultCredentials(context.Background(), scopes...)
	if err != nil {
		log.Err(err).Msg("Error getting default credentials")
		return "", err
	}

	token, err := credentials.TokenSource.Token()
	if err != nil {
		log.Err(err).Msg("Error getting token")
		return "", err
	}

	return token.AccessToken, nil
}

package auth

import (
	"code-intelligence.com/cifuzz/internal/api"
	"code-intelligence.com/cifuzz/internal/cmdutils/login"
	"code-intelligence.com/cifuzz/internal/tokenstorage"
	"code-intelligence.com/cifuzz/pkg/dialog"
	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/messaging"
)

// GetAuthStatus returns the authentication status of the user
// for the given server
func GetAuthStatus(server string) (bool, error) {
	// Obtain the API access token
	token := login.GetToken(server)

	if token == "" {
		return false, nil
	}

	// Token might be invalid, so try to authenticate with it
	apiClient := api.APIClient{Server: server}
	err := login.CheckValidToken(&apiClient, token)
	if err != nil {
		log.Warnf(`Failed to authenticate with the configured API access token.
It's possible that the token has been revoked. Please try again after
removing the token from %s.`, tokenstorage.GetTokenFilePath())

		return false, err
	}

	return true, nil
}

// ShowServerConnectionDialog ask users if they want to use a SaaS backend
// if they are not authenticated and returns their wish to authenticate
func ShowServerConnectionDialog(server string, context messaging.MessagingContext) (bool, error) {
	additionalParams := messaging.ShowServerConnectionMessage(server, context)

	wishToAuthenticate, err := dialog.Confirm("Do you want to authenticate?", true)
	if err != nil {
		return false, err
	}

	if wishToAuthenticate {
		apiClient := api.APIClient{Server: server}
		_, err := login.ReadCheckAndStoreTokenInteractively(&apiClient, additionalParams)
		if err != nil {
			return false, err
		}
	}

	return wishToAuthenticate, nil
}

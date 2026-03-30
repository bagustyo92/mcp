package bitbucket

import (
	"encoding/base64"
	"fmt"

	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/config"
)

// buildBasicAuth creates the Base64-encoded credentials for HTTP Basic auth.
func buildBasicAuth(auth config.BitbucketAuth) (string, error) {
	var credentials string

	switch {
	case auth.Email != "" && auth.APIToken != "":
		credentials = auth.Email + ":" + auth.APIToken
	case auth.Username != "" && auth.AppPassword != "":
		credentials = auth.Username + ":" + auth.AppPassword
	default:
		return "", fmt.Errorf("invalid auth config: provide either (email + api_token) or (username + app_password)")
	}

	return base64.StdEncoding.EncodeToString([]byte(credentials)), nil
}

// getHeaders returns HTTP headers for GET requests.
func getHeaders(auth config.BitbucketAuth) (map[string]string, error) {
	encoded, err := buildBasicAuth(auth)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"Authorization": "Basic " + encoded,
		"Accept":        "application/json",
	}, nil
}

// postHeaders returns HTTP headers for POST/PUT requests.
func postHeaders(auth config.BitbucketAuth) (map[string]string, error) {
	headers, err := getHeaders(auth)
	if err != nil {
		return nil, err
	}
	headers["Content-Type"] = "application/json"
	return headers, nil
}

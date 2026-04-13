package bitbucket

import (
	"encoding/base64"
	"fmt"

	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/config"
)

func buildBasicAuth(auth config.BitbucketAuth) string {
	var user, pass string
	if auth.Email != "" && auth.APIToken != "" {
		user = auth.Email
		pass = auth.APIToken
	} else {
		user = auth.Username
		pass = auth.AppPassword
	}
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", user, pass)))
}

func getHeaders(auth config.BitbucketAuth) map[string]string {
	return map[string]string{
		"Authorization": "Basic " + buildBasicAuth(auth),
		"Accept":        "application/json",
	}
}

func postHeaders(auth config.BitbucketAuth) map[string]string {
	return map[string]string{
		"Authorization": "Basic " + buildBasicAuth(auth),
		"Accept":        "application/json",
		"Content-Type":  "application/json",
	}
}

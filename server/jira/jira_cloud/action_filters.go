package jira_cloud

import (
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/lib"
)

func RequireJiraCloudUpstream(a action.Action) error {
	err := lib.RequireUpstream(a)
	if err != nil {
		return err
	}
	cloudUp, ok := a.Context().Upstream.(*jiraCloudUpstream)
	if !ok {
		return a.RespondError(http.StatusInternalServerError, errors.Errorf(
			"Jira Cloud upstream required, got %T", a.Context().Upstream))
	}
	a.Debugf("action: verified Jira Cloud instance %q", cloudUp.Config().Key)
	return nil
}

func RequireJiraCloudJWT(a action.Action) error {
	ac := a.Context()
	if ac.UpstreamJWT != nil {
		return nil
	}
	err := RequireJiraCloudUpstream(a)
	if err != nil {
		return err
	}
	cloudUp, _ := ac.Upstream.(*jiraCloudUpstream)

	tokenString := a.FormValue("jwt")
	if tokenString == "" {
		return a.RespondError(http.StatusBadRequest, nil,
			"no jwt found in the HTTP request")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.Errorf(
				"unsupported signing method: %v", token.Header["alg"])
		}
		// HMAC secret is a []byte
		return []byte(cloudUp.atlassianSecurityContext.SharedSecret), nil
	})
	if err != nil || !token.Valid {
		return a.RespondError(http.StatusUnauthorized, err,
			"failed to validate JWT")
	}

	ac.UpstreamJWT = token
	ac.UpstreamRawJWT = tokenString
	a.Debugf("action: verified Jira JWT")
	return nil
}

func RequireJiraClient(a action.Action) error {
	ac := a.Context()
	if ac.JiraClient != nil {
		return nil
	}
	err := action.Script{lib.RequireUpstream, lib.RequireBackendUser}.Run(a)
	if err != nil {
		return err
	}

	jiraClient, err := ac.Upstream.GetClient(ac.PluginId, ac.User)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	ac.JiraClient = jiraClient
	a.Debugf("action: loaded Jira client for %q", ac.User.UpstreamDisplayName())
	return nil
}

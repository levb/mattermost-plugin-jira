package jira_cloud

import (
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/lib"
)

func RequireUpstream(a action.Action) error {
	err := lib.RequireUpstream(a)
	if err != nil {
		return err
	}
	up, ok := a.Context().Upstream.(*JiraCloudUpstream)
	if !ok {
		return a.RespondError(http.StatusInternalServerError, errors.Errorf(
			"Jira Cloud upstream required, got %T", a.Context().Upstream))
	}
	a.Debugf("action: verified Jira Cloud instance %+v", up)
	return nil
}

func RequireJWT(a action.Action) error {
	ac := a.Context()
	if ac.UpstreamJWT != nil {
		return nil
	}
	err := RequireUpstream(a)
	if err != nil {
		return err
	}
	cloudUp, _ := ac.Upstream.(*JiraCloudUpstream)

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

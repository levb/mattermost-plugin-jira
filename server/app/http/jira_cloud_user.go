// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package http

import (
	"errors"
	"net/http"
	"path"

	"github.com/mattermost/mattermost-plugin-jira/server/app"
	"github.com/mattermost/mattermost-plugin-jira/server/filters"

	"github.com/mattermost/mattermost-plugin-jira/server/instance/jira_cloud"

	"github.com/andygrunwald/go-jira"
	"github.com/dgrijalva/jwt-go"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/store"
	"github.com/mattermost/mattermost-server/model"
)

const (
	argJiraJWT = "jwt"
	argMMToken = "mm_token"
)

func connectJiraCloudUser(a action.Action) error {
	ac := a.Context()
	request, err := action.HTTPRequest(a)
	if err != nil {
		return err
	}
	cloudInstance, ok := ac.Instance.(*jira_cloud.Instance)
	if !ok {
		return a.RespondError(http.StatusInternalServerError, nil, "misconfigured instance type")
	}

	mmtoken := a.FormValue(argMMToken)
	user, secret, status, err := parseTokens(cloudInstance, ac.BackendJWT, mmtoken, ac.MattermostUserId)
	if err != nil {
		return a.RespondError(status, err, "failed to parse tokens")
	}

	switch request.URL.Path {
	case routeJiraCloudUserConnected:
		var storedSecret []byte
		storedSecret, err = ac.OneTimeStore.Load(ac.MattermostUserId)
		if err != nil {
			return a.RespondError(http.StatusUnauthorized, err, "failed to confirm the link")
		}
		if len(storedSecret) == 0 || string(storedSecret) != secret {
			return a.RespondError(http.StatusUnauthorized, nil, "this link has already been used")
		}
		err = app.StoreUserNotify(ac.API, ac.UserStore, ac.Instance, ac.MattermostUserId, user)
		a.Debugf("Stored and notified: %s %+v", ac.MattermostUserId, ac.User)

	case routeJiraCloudUserDisconnected:
		err = filters.RequireBackendUser(a)
		if err != nil {
			return err
		}
		err = app.DeleteUserNotify(ac.API, ac.UserStore, ac.MattermostUserId)
		a.Debugf("Deleted and notified: %s %+v", ac.MattermostUserId, ac.User)

	case routeJiraCloudUserConfirm:
	}
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err, "(dis)connect failed")
	}

	// This set of props should work for all relevant routes/templates
	return a.RespondTemplate(request.URL.Path, "text/html", struct {
		ConnectSubmitURL      string
		DisconnectSubmitURL   string
		ArgJiraJWT            string
		ArgMMToken            string
		MMToken               string
		JiraDisplayName       string
		MattermostDisplayName string
	}{
		DisconnectSubmitURL:   path.Join(ac.PluginURLPath, routeJiraCloudUserDisconnected),
		ConnectSubmitURL:      path.Join(ac.PluginURLPath, routeJiraCloudUserConnected),
		ArgJiraJWT:            argJiraJWT,
		ArgMMToken:            argMMToken,
		MMToken:               mmtoken,
		JiraDisplayName:       user.DisplayName + " (" + user.Name + ")",
		MattermostDisplayName: ac.MattermostUser.GetDisplayName(model.SHOW_NICKNAME_FULLNAME),
	})
}

func parseTokens(cloudInstance *jira_cloud.Instance,
	backendJWT *jwt.Token, mmtoken, mattermostUserId string) (
	user *store.User, secret string, status int, err error) {

	claims, ok := backendJWT.Claims.(jwt.MapClaims)
	if !ok {
		return nil, "", http.StatusBadRequest, errors.New("invalid JWT claims")
	}
	contextClaim, ok := claims["context"].(map[string]interface{})
	if !ok {
		return nil, "", http.StatusBadRequest, errors.New("invalid JWT claim context")
	}
	userProps, ok := contextClaim["user"].(map[string]interface{})
	if !ok {
		return nil, "", http.StatusBadRequest, errors.New("invalid JWT: no user data")
	}
	userKey, _ := userProps["userKey"].(string)
	username, _ := userProps["username"].(string)
	displayName, _ := userProps["displayName"].(string)
	user = &store.User{
		User: jira.User{
			Key:         userKey,
			Name:        username,
			DisplayName: displayName,
		},
	}

	requestedUserId, secret, err := cloudInstance.ParseAuthToken(mmtoken)
	if err != nil {
		return nil, "", http.StatusUnauthorized, err
	}

	if mattermostUserId != requestedUserId {
		return nil, "", http.StatusUnauthorized, errors.New("not authorized, user id does not match link")
	}

	return user, secret, http.StatusOK, nil
}

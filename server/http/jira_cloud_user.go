// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package http

import (
	"errors"
	"net/http"
	"path"

	"github.com/dgrijalva/jwt-go"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/http_action"
	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/jira/jira_cloud"
	"github.com/mattermost/mattermost-plugin-jira/server/lib"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/model"
)

const (
	argJiraJWT = "jwt"
	argMMToken = "mm_token"
)

func connectJiraCloudUser(a action.Action) error {
	ac := a.Context()
	// up := ac.Upstream.(*jira_cloud.JiraCloudUpstream)
	request, err := http_action.Request(a)
	if err != nil {
		return err
	}

	mmtoken := a.FormValue(argMMToken)

	user, secret, status, err := parseTokens(ac.Upstream, ac.UpstreamJWT, mmtoken, ac.MattermostUserId)
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
		err = lib.StoreUserNotify(ac.API, ac.Upstream, user)
		a.Debugf("Stored and notified: %s %+v", ac.MattermostUserId, ac.User)

	case routeJiraCloudUserDisconnected:
		err = lib.RequireUpstreamUser(a)
		if err != nil {
			return err
		}
		err = lib.DeleteUserNotify(ac.API, ac.Upstream, ac.User)
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
		JiraDisplayName:       user.UpstreamDisplayName(),
		MattermostDisplayName: ac.MattermostUser.GetDisplayName(model.SHOW_NICKNAME_FULLNAME),
	})
}

func parseTokens(up upstream.Upstream, upstreamJWT *jwt.Token,
	mmtoken, mattermostUserId string) (upstream.User, string, int, error) {

	claims, ok := upstreamJWT.Claims.(jwt.MapClaims)
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
	juser := jira.JiraUser{
		Key:         userKey,
		Name:        username,
		DisplayName: displayName,
	}

	cloudUp := up.(*jira_cloud.JiraCloudUpstream)
	requestedUserId, secret, err := cloudUp.ParseAuthToken(mmtoken)
	if err != nil {
		return nil, "", http.StatusUnauthorized, err
	}

	if mattermostUserId != requestedUserId {
		return nil, "", http.StatusUnauthorized, errors.New("not authorized, user id does not match link")
	}

	user := jira.User{
		User:     upstream.NewUser(mattermostUserId, userKey),
		JiraUser: juser,
	}

	return user, secret, http.StatusOK, nil
}

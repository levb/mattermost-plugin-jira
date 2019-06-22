// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package http

import (
	"net/http"
	"path"

	"github.com/dghubble/oauth1"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/app"
	"github.com/mattermost/mattermost-plugin-jira/server/filters"
	"github.com/mattermost/mattermost-plugin-jira/server/instance/jira_server"
	"github.com/mattermost/mattermost-plugin-jira/server/store"
	"github.com/mattermost/mattermost-server/model"
)

func completeJiraServerOAuth1(a action.Action) error {
	ac := a.Context()
	request, err := action.HTTPRequest(a)
	if err != nil {
		return err
	}

	requestToken, verifier, err := oauth1.ParseAuthorizationCallback(request)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to parse callback request from Jira")
	}

	oauthTmpCredentials, err := ac.OneTimeStore.LoadOauth1aTemporaryCredentials(ac.MattermostUserId)
	if err != nil || oauthTmpCredentials == nil || len(oauthTmpCredentials.Token) <= 0 {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to get temporary credentials for %q", ac.MattermostUserId)
	}

	if oauthTmpCredentials.Token != requestToken {
		return a.RespondError(http.StatusUnauthorized, nil, "request token mismatch")
	}

	serverInstance, ok := ac.Instance.(*jira_server.Instance)
	if !ok {
		return a.RespondError(http.StatusInternalServerError, nil, "misconfiguration, wrong Action type")
	}

	oauth1Config, err := serverInstance.GetOAuth1Config(ac.PluginURL)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to obtain oauth1 config")
	}

	// Although we pass the oauthTmpCredentials as required here. The Jira server does not appar to validate it.
	// We perform the check above for reuse so this is irrelavent to the security from our end.
	accessToken, accessSecret, err := oauth1Config.AccessToken(requestToken, oauthTmpCredentials.Secret, verifier)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err,
			"failed to obtain oauth1 access token")
	}

	user := &store.User{
		Oauth1AccessToken:  accessToken,
		Oauth1AccessSecret: accessSecret,
	}
	jiraClient, err := serverInstance.GetClient(ac.PluginURL, user)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	juser, _, err := jiraClient.User.GetSelf()
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	user.User = *juser
	// Set default settings the first time a user connects
	user.Settings = &store.UserSettings{Notifications: true}

	err = app.StoreUserNotify(ac.API, ac.UserStore, ac.Instance, ac.MattermostUserId, user)
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err)
	}
	a.Debugf("Stored and notified: %s %+v", ac.MattermostUserId, user)

	return a.RespondTemplate(request.URL.Path, "text/html", struct {
		MattermostDisplayName string
		JiraDisplayName       string
		RevokeURL             string
	}{
		JiraDisplayName:       juser.DisplayName + " (" + juser.Name + ")",
		MattermostDisplayName: ac.MattermostUser.GetDisplayName(model.SHOW_NICKNAME_FULLNAME),
		RevokeURL:             path.Join(ac.PluginURLPath, routeUserDisconnect),
	})
}

func getJiraServerOAuth1PublicKey(a action.Action) error {
	err := action.Script{
		filters.RequireHTTPGet,
		filters.RequireInstance,
		filters.RequireMattermostUserId,
	}.Run(a)
	if err != nil {
		return err
	}
	ac := a.Context()
	serverInstance, ok := ac.Instance.(*jira_server.Instance)
	if !ok {
		return a.RespondError(http.StatusInternalServerError, nil, "misconfigured instance type")
	}

	if !ac.API.HasPermissionTo(ac.MattermostUserId, model.PERMISSION_MANAGE_SYSTEM) {
		return a.RespondError(http.StatusForbidden, nil, "forbidden")
	}

	pkey, err := serverInstance.PublicKeyString()
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err, "failed to load public key")
	}
	return a.RespondPrintf(string(pkey))
}

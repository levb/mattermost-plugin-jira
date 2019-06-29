// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira_server

import (
	"net/http"

	"github.com/dghubble/oauth1"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/lib"
	"github.com/mattermost/mattermost-plugin-jira/server/store"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/plugin"
)

func (serverUp JiraServerUpstream) CompleteOAuth1(api plugin.API, ots store.OneTimeStore,
	r *http.Request, pluginURL, mattermostUserId string) (upstream.User, int, error) {

	requestToken, verifier, err := oauth1.ParseAuthorizationCallback(r)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.WithMessage(err,
			"failed to parse callback request from Jira")
	}

	oauthTmpCredentials, err := ots.LoadOauth1aTemporaryCredentials(mattermostUserId)
	if err != nil || oauthTmpCredentials == nil || len(oauthTmpCredentials.Token) <= 0 {
		return nil, http.StatusInternalServerError, errors.WithMessagef(err,
			"failed to get temporary credentials for %q", mattermostUserId)
	}

	if oauthTmpCredentials.Token != requestToken {
		return nil, http.StatusUnauthorized, errors.New("request token mismatch")
	}

	oauth1Config, err := serverUp.GetOAuth1Config(pluginURL)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.WithMessage(err,
			"failed to obtain oauth1 config")
	}

	// Although we pass the oauthTmpCredentials as required here. The Jira server does not appar to validate it.
	// We perform the check above for reuse so this is irrelavent to the security from our end.
	accessToken, accessSecret, err := oauth1Config.AccessToken(requestToken, oauthTmpCredentials.Secret, verifier)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.WithMessage(err,
			"failed to obtain oauth1 access token")
	}

	// We don't have the Jira user info yet, but have enough to obtain the client
	user := &jiraServerUser{
		User: jira.User{
			User: upstream.NewUser(mattermostUserId, ""),
		},
		Oauth1AccessToken:  accessToken,
		Oauth1AccessSecret: accessSecret,
	}
	jiraClient, err := serverUp.GetClient(pluginURL, user)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	juser, _, err := jiraClient.User.GetSelf()
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	user.User.JiraUser = jira.JiraUser(*juser)
	user.User.User = upstream.NewUser(mattermostUserId, juser.Key)

	// Set default settings the first time a user connects
	*(user.Settings()) = upstream.UserSettings{
		Notifications: true,
	}

	err = lib.StoreUserNotify(api, serverUp, user)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return user, http.StatusOK, nil
}

// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira_server

import (
	"net/http"

	"github.com/dghubble/oauth1"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/proxy"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/plugin"
)

type user struct {
	jira.User
	Oauth1AccessToken  string `json:",omitempty"`
	Oauth1AccessSecret string `json:",omitempty"`
}

type oauth1aTempCredentials struct {
	Token  string
	Secret string
}

func (u user) UpstreamUserId() string {
	return u.JiraUser.Name
}

func (serverUp *serverUpstream) completeOAuth1(api plugin.API, ots kvstore.OneTimeStore,
	r *http.Request, pluginURL, mattermostUserId string) (upstream.User, int, error) {

	requestToken, verifier, err := oauth1.ParseAuthorizationCallback(r)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.WithMessage(err,
			"failed to parse callback request from Jira")
	}

	oauthTmpCredentials, err := loadTempCredentials(ots, mattermostUserId)
	if err != nil || oauthTmpCredentials == nil || len(oauthTmpCredentials.Token) <= 0 {
		return nil, http.StatusInternalServerError, errors.WithMessagef(err,
			"failed to get temporary credentials for %q", mattermostUserId)
	}

	if oauthTmpCredentials.Token != requestToken {
		return nil, http.StatusUnauthorized, errors.New("request token mismatch")
	}

	oauth1Config, err := serverUp.getOAuth1Config(pluginURL)
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
	user := &user{
		User: jira.User{
			BasicUser: *upstream.NewBasicUser(mattermostUserId, ""),
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
	user.User.BasicUser.UUserId = juser.Name

	err = proxy.StoreUserNotify(api, serverUp, user)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return user, http.StatusOK, nil
}

func storeTempCredentials(ots kvstore.OneTimeStore, mmUserId string, credentials *oauth1aTempCredentials) error {
	s := kvstore.NewHashedKeyStore(ots, kvstore.KeyPrefixTempOAuth1aCredentials)
	return kvstore.StoreJSON(s, mmUserId, credentials)
}

func loadTempCredentials(ots kvstore.OneTimeStore, mmUserId string) (*oauth1aTempCredentials, error) {
	s := kvstore.NewHashedKeyStore(ots, kvstore.KeyPrefixTempOAuth1aCredentials)
	var credentials oauth1aTempCredentials
	err := kvstore.LoadJSON(s, mmUserId, &credentials)
	if err != nil {
		return nil, err
	}
	return &credentials, nil
}

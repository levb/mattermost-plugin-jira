// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package http

import (
	"net/http"
	"path"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/http_action"
	"github.com/mattermost/mattermost-plugin-jira/server/jira/jira_server"
)

func completeJiraServerOAuth1(a action.Action) error {
	ac := a.Context()
	r, err := http_action.Request(a)
	if err != nil {
		return err
	}
	up, _ := ac.Upstream.(*jira_server.JiraServerUpstream)

	u, status, err := up.CompleteOAuth1(ac.API, ac.OneTimeStore, r, ac.PluginURL, ac.MattermostUserId)
	if err != nil {
		a.RespondError(status, err)
	}

	return http_action.RespondTemplate(a, "text/html", struct {
		MattermostDisplayName string
		JiraDisplayName       string
		RevokeURL             string
	}{
		JiraDisplayName:       u.UpstreamDisplayName(),
		MattermostDisplayName: u.MattermostDisplayName(),
		RevokeURL:             path.Join(ac.PluginURLPath, routeUserDisconnect),
	})
}

func getJiraServerOAuth1PublicKey(a action.Action) error {
	ac := a.Context()
	serverUp := ac.Upstream.(*jira_server.JiraServerUpstream)
	pkey, err := serverUp.PublicKeyString()
	if err != nil {
		return a.RespondError(http.StatusInternalServerError, err, "failed to load public key")
	}
	return a.RespondPrintf(string(pkey))
}

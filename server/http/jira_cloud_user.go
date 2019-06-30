// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package http

import (
	"path"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/http_action"
	"github.com/mattermost/mattermost-plugin-jira/server/jira/jira_cloud"
	"github.com/mattermost/mattermost-plugin-jira/server/lib"
	"github.com/mattermost/mattermost-server/model"
)

const (
	argJiraJWT = "jwt"
	argMMToken = "mm_token"
)

func connectJiraCloudUser(a action.Action) error {
	ac := a.Context()
	// up := ac.Upstream.(*jira_cloud.JiraCloudUpstream)
	request := http_action.Request(a)
	mmtoken := a.FormValue(argMMToken)

	user, secret, status, err := jira_cloud.ParseTokens(ac.Upstream, ac.UpstreamJWT,
		mmtoken, ac.MattermostUserId)
	if err != nil {
		return a.RespondError(status, err, "failed to parse tokens")
	}

	switch request.URL.Path {
	case routeJiraCloudUserConnected:
		status, err := jira_cloud.ProcessUserConnected(ac.API, ac.Upstream, ac.OneTimeStore,
			user, secret, ac.MattermostUserId)
		if err != nil {
			a.RespondError(status, err)
		}
		a.Debugf("Stored and notified: %s %+v", ac.MattermostUserId, ac.User)

	case routeJiraCloudUserDisconnected:
		err := lib.RequireUpstreamUser(a)
		if err != nil {
			return err
		}
		status, err := jira_cloud.ProcessUserDisconnected(ac.API, ac.Upstream, ac.User)
		if err != nil {
			a.RespondError(status, err)
		}
		a.Debugf("Deleted and notified: %s %+v", ac.MattermostUserId, ac.User)

	case routeJiraCloudUserConfirm:
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

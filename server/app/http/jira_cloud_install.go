// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package http

import (
	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/app"
	"github.com/mattermost/mattermost-plugin-jira/server/instance/jira_cloud"
)

func getJiraCloudInstallJSON(a action.Action) error {
	ac := a.Context()
	return action.HTTPRespondTemplate(a,
		"application/json",
		map[string]string{
			"BaseURL":            ac.PluginURL,
			"RouteACJSON":        routeJiraCloudInstallJSON,
			"RouteACInstalled":   routeJiraCloudInstalled,
			"RouteACUninstalled": routeJiraCloudUninstalled,
			"RouteACUserConfirm": routeJiraCloudUserConfirm,
			"UserLandingPageKey": jira_cloud.UserLandingPageKey,
			"ExternalURL":        ac.MattermostSiteURL,
			"PluginKey":          ac.PluginKey,
		})
}

func processJiraCloudInstalled(a action.Action) error {
	ac := a.Context()
	request, err := action.HTTPRequest(a)
	if err != nil {
		return err
	}

	status, err := app.ProcessJiraCloudInstalled(ac.API,
		ac.InstanceStore, ac.CurrentInstanceStore, ac.AuthTokenSecret, request.Body)
	if err != nil {
		return a.RespondError(status, err,
			"failed to process atlassian-connect installed event")
	}

	return a.RespondJSON([]string{"OK"})
}

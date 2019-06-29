// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package http

import (
	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/http_action"
	"github.com/mattermost/mattermost-plugin-jira/server/jira/jira_cloud"
)

func getJiraCloudInstallJSON(a action.Action) error {
	ac := a.Context()
	return http_action.RespondTemplate(a,
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
	r, err := http_action.Request(a)
	if err != nil {
		return err
	}

	status, err := jira_cloud.ProcessInstalled(ac.API, ac.UpstreamStore,
		ac.AuthTokenSecret, r.Body)
	if err != nil {
		return a.RespondError(status, err,
			"failed to process atlassian-connect installed event")
	}

	return a.RespondJSON([]string{"OK"})
}
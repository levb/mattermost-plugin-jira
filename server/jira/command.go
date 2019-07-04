// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira

import (
	"strings"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/command_action"
	"github.com/mattermost/mattermost-plugin-jira/server/proxy"
)

var CommandRoutes = map[string]*action.Route{
	"transition": action.NewRoute(
		RequireClient,
		commandTransition).With(
		&command_action.Metadata{MinArgc: 2, MaxArgc: -1, ArgNames: []string{"key"}}),
	"webhook/url": action.NewRoute(
		proxy.RequireMattermostSysAdmin,
		commandWebhookURL).With(
		&command_action.Metadata{MinArgc: 0, MaxArgc: 0}),
}

func commandWebhookURL(a action.Action) error {
	ac := a.Context()
	u, err := getWebhookURL(ac.API, ac.PluginURL, ac.WebhookSecret, ac.MattermostTeamId, ac.MattermostChannelId)
	if err != nil {
		return a.RespondError(0, err)
	}
	return a.RespondPrintf("Please use the following URL to set up a Jira webhook: %v", u)
}

func commandTransition(a action.Action) error {
	ac := a.Context()
	issueKey := a.FormValue("key")
	toState := strings.Join(command_action.Argv(a)[1:], " ")

	msg, err := transitionIssue(ac.JiraClient, ac.Upstream, issueKey, toState)
	if err != nil {
		return a.RespondError(0, err)
	}
	return a.RespondPrintf(msg)
}

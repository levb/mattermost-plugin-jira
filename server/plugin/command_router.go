// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package plugin

import (
	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/command_action"
	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/proxy"
)

var commandRouter = &action.Router{
	Before:  action.Script{command_action.LogAction},
	Default: help,

	// MattermostUserID is set for all commands, so no special "Requir" for it
	Routes: map[string]*action.Route{
		"connect":    action.NewRoute(proxy.RequireUpstream, connect),
		"disconnect": action.NewRoute(proxy.RequireUpstreamUser, disconnect),
		"settings/notifications/": action.NewRoute(jira.RequireClient, notifications).With(
			&command_action.Metadata{MinArgc: 1, MaxArgc: 1,
				ArgNames: []string{"value"}}),
		"upstream/list": action.NewRoute(proxy.RequireMattermostSysAdmin, list),
		"upstream/select": action.NewRoute(proxy.RequireMattermostSysAdmin, selectUpstream).With(
			&command_action.Metadata{MinArgc: 1, MaxArgc: 1, ArgNames: []string{"n"}}),
		"upstream/delete": action.NewRoute(proxy.RequireMattermostSysAdmin, deleteUpstream).With(
			&command_action.Metadata{MinArgc: 1, MaxArgc: 1, ArgNames: []string{"n"}}),
	},
	// 	RequireMattermostSysAdmin,
	// "transition": {
	// 	Handler:  commandTransition,
	// 	Metadata: &action.CommandMetadata{MinArgc: 2, MaxArgc: -1, ArgNames: []string{"key"}},
	// },
	// "install/server": {
	// 	Handler:  commandInstallServer,
	// 	Metadata: &action.CommandMetadata{MinArgc: 1, MaxArgc: 1, ArgNames: []string{"url"}},
	// },
	// "install/cloud": {
	// 	Handler:  commandInstallCloud,
	// 	Metadata: &action.CommandMetadata{MinArgc: 1, MaxArgc: 1, ArgNames: []string{"url"}},
	// },
	// "uninstall/server": {
	// 	Handler:  commandUninstallServer,
	// 	Metadata: &action.CommandMetadata{MinArgc: 1, MaxArgc: 1, ArgNames: []string{"url"}},
	// },
	// "uninstall/cloud": {
	// 	Handler:  commandUninstallCloud,
	// 	Metadata: &action.CommandMetadata{MinArgc: 1, MaxArgc: 1, ArgNames: []string{"url"}},
	// },
	// // used for debugging, uncomment if needed
	// "webhook":         commandWebhookURL,
}

func help(a action.Action) error {
	return a.RespondPrintf(helpText)
}

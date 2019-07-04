// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package plugin

import (
	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/command_action"
	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/jira/jira_cloud"
	"github.com/mattermost/mattermost-plugin-jira/server/jira/jira_server"
	"github.com/mattermost/mattermost-plugin-jira/server/proxy"
)

func init() {
	// Remove this comment to enable debug commands
	commandRouter.AddRoutes(debugRoutes)

	commandRouter.AddRoutes(commonRoutes)
	commandRouter.AddRoutes(jira.CommandRoutes)
	commandRouter.AddRoutes(jira_server.CommandRoutes)
	commandRouter.AddRoutes(jira_cloud.CommandRoutes)
}

var commandRouter = &action.Router{
	Before:  action.Script{command_action.RequireCommandAction},
	After:   action.Script{command_action.LogAction},
	Default: help,
	Routes:  map[string]*action.Route{},
}

var commonRoutes = map[string]*action.Route{
	"connect": action.NewRoute(
		proxy.RequireUpstream,
		commandConnect),
	"disconnect": action.NewRoute(
		proxy.RequireUpstreamUser,
		commandDisconnect),
	"uninstall": action.NewRoute(
		proxy.RequireUpstream,
		proxy.RequireMattermostSysAdmin,
		commandUpstreamUninstall).With(
		&command_action.Metadata{MinArgc: 2, MaxArgc: 2, ArgNames: []string{"type", "key"}}),
	"settings/notifications": action.NewRoute(
		jira.RequireClient,
		commandSettingsNotifications).With(
		&command_action.Metadata{MinArgc: 1, MaxArgc: 1, ArgNames: []string{"value"}}),
}

var debugRoutes = map[string]*action.Route{
	"debug/upstream/list": action.NewRoute(
		proxy.RequireMattermostSysAdmin,
		commandUpstreamList),
	"debug/upstream/select": action.NewRoute(
		proxy.RequireMattermostSysAdmin,
		commandUpstreamSelect).With(
		&command_action.Metadata{MinArgc: 1, MaxArgc: 1, ArgNames: []string{"key"}}),
}

func help(a action.Action) error {
	const helpText = "###### Mattermost Jira Plugin - Slash Command Help\n" +
		"* `/jira connect` - Connect your Mattermost account to your Jira account\n" +
		"* `/jira disconnect` - Disconnect your Mattermost account from your Jira account\n" +
		"* `/jira create <text (optional)>` - Create a new Issue with 'text' inserted into the description field.\n" +
		"* `/jira transition <issue-key> <state>` - Change the state of a Jira issue\n" +
		"* `/jira settings [setting] [value]` - Update your user settings\n" +
		"  * [setting] can be `notifications`\n" +
		"  * [value] can be `on` or `off`\n" +

		"\n###### For System Administrators:\n" +
		"Install:\n" +
		"* `/jira install cloud <URL>` - Connect Mattermost to a Jira Cloud instance located at <URL>\n" +
		"* `/jira install server <URL>` - Connect Mattermost to a Jira Server or Data Center instance located at <URL>\n" +
		"Uninstall:\n" +
		"* `/jira uninstall cloud <URL>` - Disconnect Mattermost from a Jira Cloud instance located at <URL>\n" +
		"* `/jira uninstall server <URL>` - Disconnect Mattermost from a Jira Server or Data Center instance located at <URL>\n" +
		""
	return a.RespondPrintf(helpText)
}

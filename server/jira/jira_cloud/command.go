// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira_cloud

import (
	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/command_action"
	"github.com/mattermost/mattermost-plugin-jira/server/proxy"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
)

var CommandRoutes = map[string]*action.Route{
	"install/cloud": action.NewRoute(
		proxy.RequireMattermostSysAdmin,
		commandInstallCloud).With(
		&command_action.Metadata{MinArgc: 1, MaxArgc: 1, ArgNames: []string{"key"}}),
}

func commandInstallCloud(a action.Action) error {
	ac := a.Context()
	jiraURL := a.FormValue("key")
	jiraURL, err := upstream.NormalizeURL(jiraURL)
	if err != nil {
		return a.RespondError(0, err, "invalid Jira URL")
	}

	err = storeUnconfirmedUpstream(ac.OneTimeStore, jiraURL)
	if err != nil {
		return a.RespondError(0, err)
	}

	const addResponseFormat = `
%s has been successfully installed. To finish the configuration, create a new app in your Jira instance following these steps:

1. Navigate to [**Settings > Apps > Manage Apps**](%s/plugins/servlet/upm?source=side_nav_manage_addons).
  - For older versions of Jira, navigate to **Administration > Applications > Add-ons > Manage add-ons**.
2. Click **Settings** at bottom of page, enable development mode, and apply this change.
  - Enabling development mode allows you to install apps that are not from the Atlassian Marketplace.
3. Click **Upload app**.
4. In the **From this URL field**, enter: ` + "`%s%s`" + `
5. Wait for the app to install. Once completed, you should see an "Installed and ready to go!" message.
6. Use the ` + "`/jira connect`" + `command to connect your Mattermost account with your Jira account.
7. Click the **More Actions** (...) option of any message in the channel (available when you hover over a message).

If you see an option to create a Jira issue, you're all set! If not, refer to our [documentation](https://about.mattermost.com/default-jira-plugin) for troubleshooting help.
`

	// TODO What is the exact group membership in Jira required? Site-admins?
	return a.RespondPrintf(addResponseFormat, jiraURL, jiraURL, ac.PluginURL, routeInstallJSON)
}

// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira_server

import (
	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/command_action"
	"github.com/mattermost/mattermost-plugin-jira/server/proxy"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
)

var CommandRoutes = map[string]*action.Route{
	"install/server": action.NewRoute(
		proxy.RequireMattermostSysAdmin,
		commandInstallServer).With(
		&command_action.Metadata{MinArgc: 1, MaxArgc: 1, ArgNames: []string{"key"}}),
}

func commandInstallServer(a action.Action) error {
	ac := a.Context()
	jiraURL := a.FormValue("key")
	jiraURL, err := upstream.NormalizeURL(jiraURL)
	if err != nil {
		return a.RespondError(0, err)
	}

	up := newUpstream(ac.UpstreamStore, jiraURL)
	err = ac.UpstreamStore.StoreUpstream(up)
	if err != nil {
		return a.RespondError(0, err)
	}
	err = ac.UpstreamStore.StoreCurrentUpstream(up)
	if err != nil {
		return a.RespondError(0, err)
	}

	pkey, err := publicKeyString(up)
	if err != nil {
		return a.RespondError(0, err)
	}

	return a.RespondPrintf(
		"Server instance has been installed. To finish the configuration, add an Application Link in your Jira instance following these steps:\n\n"+
			"1. Navigate to [**Settings > Applications > Application Links**](%s/plugins/servlet/applinks/listApplicationLinks)\n"+
			"2. Enter `%s` as the application link, then click **Create new link**.\n"+
			`3. In **Configure Application URL** screen, confirm your Mattermost URL is entered as the "New URL". Ignore any displayed errors and click **Continue**.`+"\n"+
			`4. In **Link Applications** screen, set the following values:`+"\n"+
			"- **Application Name**: `Mattermost` (or your choice)\n"+
			"- **Application Type**: `Generic Application`\n"+
			"5. Check the **Create incoming link** value, then click **Continue**.\n"+
			"6. In the following **Link Applications** screen, set the following values:\n"+
			"- **Consumer Key**: `%s`\n"+
			"- **Consumer Name**: `Mattermost` (or your choice)"+
			"- **Public Key**:\n```%s```\n"+
			"7. Click **Continue**.\n"+
			"8. Use the `/jira connect` command to connect your Mattermost account with your Jira account.\n"+
			"9. Click the **More Actions** (...) option of any message in the channel (available when you hover over a message).\n\n"+
			"If you see an option to create a Jira issue, you're all set! If not, refer to our [documentation](https://about.mattermost.com/default-jira-plugin) for troubleshooting help.\n",
		up.URL(), ac.PluginSiteURL, ac.PluginKey, pkey)
}

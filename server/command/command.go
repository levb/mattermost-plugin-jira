// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package command

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/command_action"
	"github.com/mattermost/mattermost-plugin-jira/server/lib"
)

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

var Router = &action.Router{
	Before:  action.Script{command_action.LogAction},
	Default: help,

	// MattermostUserID is set for all commands, so no special "Requir" for it
	Routes: map[string]*action.Route{
		"connect":    action.NewRoute(lib.RequireUpstream, connect),
		"disconnect": action.NewRoute(lib.RequireUpstreamUser, disconnect),
		"settings/notifications/": action.NewRoute(lib.RequireUpstreamClient, notifications).With(
			&command_action.Metadata{MinArgc: 1, MaxArgc: 1,
				ArgNames: []string{"value"}}),
		"upstream/list": action.NewRoute(lib.RequireMattermostSysAdmin, list),
		"upstream/select": action.NewRoute(lib.RequireMattermostSysAdmin, selectUpstream).With(
			&command_action.Metadata{MinArgc: 1, MaxArgc: 1, ArgNames: []string{"n"}}),
		"upstream/delete": action.NewRoute(lib.RequireMattermostSysAdmin, deleteUpstream).With(
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

func connect(a action.Action) error {
	ac := a.Context()
	redirectURL, err := ac.Upstream.GetUserConnectURL(ac.OneTimeStore, ac.PluginURL, ac.MattermostUserId)
	if err != nil {
		return a.RespondError(0, err, "command failed, please contact your system administrator")
	}
	return a.RespondRedirect(redirectURL)
}

func disconnect(a action.Action) error {
	ac := a.Context()
	err := lib.DeleteUserNotify(ac.API, ac.Upstream, ac.User)
	if err != nil {
		return a.RespondError(0, err, "Could not complete the **disconnection** request")
	}
	return a.RespondPrintf("You have successfully disconnected your Jira account (**%s**).",
		ac.User.UpstreamDisplayName())
}

const (
	settingOn  = "on"
	settingOff = "off"
)

func notifications(a action.Action) error {
	ac := a.Context()
	valueStr := a.FormValue("value")
	value := false
	switch valueStr {
	case settingOn:
		value = true
	case settingOff:
		value = false
	default:
		return a.RespondPrintf(
			"`/jira settings notifications [value]`\nInvalid value %q. Accepted values are: `on` or `off`.", valueStr)
	}
	err := lib.StoreUserSettingsNotifications(ac.Upstream, ac.User, value)
	if err != nil {
		return a.RespondError(0, err)
	}
	return a.RespondPrintf("Settings updated. Notifications %s.", valueStr)
}

func list(a action.Action) error {
	ac := a.Context()
	known, err := ac.UpstreamStore.LoadKnown()
	if err != nil {
		return a.RespondError(0, err)
	}
	if len(known) == 0 {
		return a.RespondPrintf("(none installed)\n")
	}

	// error not important here, only need to highlight thee current in the list
	currentUp, _ := ac.UpstreamStore.LoadCurrent()

	keys := []string{}
	for key := range known {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	text := "Known Jira instances (selected upstream is **bold**)\n\n| |URL|Type|\n|--|--|--|\n"
	for i, key := range keys {
		up, err := ac.UpstreamStore.Load(key)
		if err != nil {
			text += fmt.Sprintf("|%v|%s|error: %v|\n", i+1, key, err)
			continue
		}
		details := ""
		for k, v := range up.DisplayDetails() {
			details += fmt.Sprintf("%s:%s, ", k, v)
		}
		if len(details) > len(", ") {
			details = details[:len(details)-2]
		} else {
			details = up.Config().Type
		}
		format := "|%v|%s|%s|\n"
		if currentUp != nil && key == currentUp.Config().Key {
			format = "| **%v** | **%s** |%s|\n"
		}
		text += fmt.Sprintf(format, i+1, key, details)
	}
	return a.RespondPrintf(text)
}

// var commandInstallCloud = ActionScript{
// 	RequireMattermostSysAdmin,
// 	executeInstallCloud,
// }

// func executeInstallCloud(a action.Action) error {
// 	if len(a.Args) != 1 {
// 		return executeHelp(a)
// 	}
// 	jiraURL := a.FormValue("$1")

// 	// Create an "uninitialized" instance of Jira Cloud that will
// 	// receive the /installed callback
// 	err := a.UpstreamStore.CreateInactiveCloudUpstream(jiraURL)
// 	if err != nil {
// 		return a.RespondError(0, err)
// 	}

// 	const addResponseFormat = `
// %s has been successfully installed. To finish the configuration, create a new app in your Jira instance following these steps:

// 1. Navigate to [**Settings > Apps > Manage Apps**](%s/plugins/servlet/upm?source=side_nav_manage_addons).
//   - For older versions of Jira, navigate to **Administration > Applications > Add-ons > Manage add-ons**.
// 2. Click **Settings** at bottom of page, enable development mode, and apply this change.
//   - Enabling development mode allows you to install apps that are not from the Atlassian Marketplace.
// 3. Click **Upload app**.
// 4. In the **From this URL field**, enter: %s%s
// 5. Wait for the app to install. Once completed, you should see an "Installed and ready to go!" message.
// 6. Use the "/jira connect" command to connect your Mattermost account with your Jira account.
// 7. Click the "More Actions" (...) option of any message in the channel (available when you hover over a message).

// If you see an option to create a Jira issue, you're all set! If not, refer to our [documentation](https://about.mattermost.com/default-jira-plugin) for troubleshooting help.
// `

// 	// TODO What is the exact group membership in Jira required? Site-admins?
// 	return a.RespondPrintf(addResponseFormat, jiraURL, jiraURL, a.PluginConfig.PluginURL, routeACJSON)
// }

// var commandInstallServer = ActionScript{
// 	RequireMattermostSysAdmin,
// 	executeInstallServer,
// }

// func executeInstallServer(a action.Action) error {
// 	if len(a.Args) != 1 {
// 		return executeHelp(a)
// 	}
// 	jiraURL := a.FormValue("$1")

// 	const addResponseFormat = `` +
// 		`Server instance has been installed. To finish the configuration, add an Application Link in your Jira instance following these steps:

// 1. Navigate to **Settings > Applications > Application Links**
// 2. Enter %s as the application link, then click **Create new link**.
// 3. In **Configure Application URL** screen, confirm your Mattermost URL is entered as the "New URL". Ignore any displayed errors and click **Continue**.
// 4. In **Link Applications** screen, set the following values:
//   - **Application Name**: Mattermost
//   - **Application Type**: Generic Application
// 5. Check the **Create incoming link** value, then click **Continue**.
// 6. In the following **Link Applications** screen, set the following values:
//   - **Consumer Key**: %s
//   - **Consumer Name**: Mattermost
//   - **Public Key**: %s
// 7. Click **Continue**.
// 6. Use the "/jira connect" command to connect your Mattermost account with your Jira account.
// 7. Click the "More Actions" (...) option of any message in the channel (available when you hover over a message).

// If you see an option to create a Jira issue, you're all set! If not, refer to our [documentation](https://about.mattermost.com/default-jira-plugin) for troubleshooting help.
// `
// 	serverUpstream := NewServerUpstream(jiraURL, a.PluginConfig.PluginKey)
// 	err := a.UpstreamStore.StoreUpstream(serverUpstream)
// 	if err != nil {
// 		return a.RespondError(0, err)
// 	}
// 	err = a.CurrentUpstreamStore.StoreCurrentUpstream(serverUpstream)
// 	if err != nil {
// 		return a.RespondError(0, err)
// 	}

// 	pkey, err := publicKeyString(a.SecretsStore)
// 	if err != nil {
// 		return a.RespondError(0, err)
// 	}
// 	return a.RespondPrintf(addResponseFormat, a.PluginConfig.SiteURL, serverUpstream.GetMattermostKey(), pkey)
// }

// var commandUninstall = ActionScript{
// 	RequireUpstream,
// 	RequireMattermostSysAdmin,
// 	executeUninstall,
// }

// // executeUninstall will uninstall the jira cloud instance if the url matches, and then update all connected
// // clients so that their Jira-related menu options are removed.
// func executeUninstall(a action.Action) error {
// 	if len(a.Args) != 1 {
// 		return executeHelp(a)
// 	}
// 	jiraURL := a.FormValue("$1")

// 	if jiraURL != a.Upstream.GetURL() {
// 		return a.RespondError(0, nil,
// 			"You have entered an incorrect URL. The current Jira instance URL is: %s. "+
// 				"Please enter the URL correctly to confirm the uninstall command.",
// 			a.Upstream.GetURL())
// 	}

// 	err := a.UpstreamStore.DeleteJiraUpstream(a.Upstream.GetURL())
// 	if err != nil {
// 		return a.RespondError(0, err,
// 			"Failed to delete Jira instance %s", a.Upstream.GetURL())
// 	}

// 	// Notify users we have uninstalled an instance
// 	a.API.PublishWebSocketEvent(
// 		wSEventUpstreamStatus,
// 		map[string]interface{}{
// 			"instance_installed": false,
// 		},
// 		&model.WebsocketBroadcast{},
// 	)

// 	const uninstallInstructions = `Jira instance successfully disconnected. Go to **Settings > Apps > Manage Apps** to remove the application in your Jira instance.`

// 	return a.RespondPrintf(uninstallInstructions)
// }

// var commandTransition = ActionScript{
// 	RequireJiraClient,
// 	executeTransition,
// }

// func executeTransition(a action.Action) error {
// 	if len(a.Args) < 2 {
// 		return executeHelp(a)
// 	}
// 	issueKey := a.FormValue("$1")
// 	toState := strings.Join(a.Args[1:], " ")

// 	msg, err := transitionJiraIssue(a, issueKey, toState)
// 	if err != nil {
// 		return a.RespondError(0, err)
// 	}
// 	return a.RespondPrintf(msg)
// }

// var commandWebhookURL = ActionScript{
// 	RequireMattermostSysAdmin,
// 	executeWebhookURL,
// }

// func executeWebhookURL(a action.Action) error {
// 	if len(a.Args) != 0 {
// 		return executeHelp(a)
// 	}

// 	u, err := GetWebhookURL(a.PluginConfig, a.API, a.CommandArgs.TeamId, a.CommandArgs.ChannelId)
// 	if err != nil {
// 		return a.RespondError(0, err)
// 	}
// 	return a.RespondPrintf("Please use the following URL to set up a Jira webhook: %v", u)
// }

// func commandResponsef(format string, args ...interface{}) *model.CommandResponse {
// 	return &model.CommandResponse{
// 		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
// 		Text:         fmt.Sprintf(format, args...),
// 		Username:     PluginMattermostUsername,
// 		IconURL:      PluginIconURL,
// 		Type:         model.POST_DEFAULT,
// 	}
// }

func selectUpstream(a action.Action) error {
	ac := a.Context()
	upkey := a.FormValue("n")
	num, err := strconv.ParseUint(upkey, 10, 8)
	if err == nil {
		known, loadErr := ac.UpstreamStore.LoadKnown()
		if loadErr != nil {
			return a.RespondError(0, err)
		}
		if num < 1 || int(num) > len(known) {
			return a.RespondError(0, nil,
				"Wrong instance number %v, must be 1-%v\n", num, len(known))
		}

		keys := []string{}
		for key := range known {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		upkey = keys[num-1]
	}

	up, err := ac.UpstreamStore.Load(upkey)
	if err != nil {
		return a.RespondError(0, err)
	}
	err = ac.UpstreamStore.StoreCurrent(up)
	if err != nil {
		return a.RespondError(0, err)
	}

	return list(a)
}

func deleteUpstream(a action.Action) error {
	ac := a.Context()
	instanceKey := a.FormValue("n")

	known, err := ac.UpstreamStore.LoadKnown()
	if err != nil {
		return a.RespondError(0, err)
	}
	if len(known) == 0 {
		return a.RespondError(0, nil,
			"There are no upstreams to delete.\n")
	}

	num, err := strconv.ParseUint(instanceKey, 10, 8)
	if err == nil {
		if num < 1 || int(num) > len(known) {
			return a.RespondError(0, nil,
				"Wrong upstream number %v, must be 1-%v\n", num, len(known)+1)
		}

		keys := []string{}
		for key := range known {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		instanceKey = keys[num-1]
	}

	// Remove the instance
	err = ac.UpstreamStore.Delete(instanceKey)
	if err != nil {
		return a.RespondError(0, err)
	}

	// if that was our only instance, just respond with an empty list.
	if len(known) == 1 {
		return list(a)
	}

	// Select instance #1
	// TODO 	return executeUpstreamSelect(a)
	return a.RespondPrintf("<><> DONE")
}

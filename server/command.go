package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
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

var commandRouter = ActionRouter{
	Log: func(a *Action) error {
		if a.LogErr != nil {
			a.Infof("command: %q error:%v", a.CommandArgs.Command, a.LogErr)
		} else {
			a.Debugf("command: %q", a.CommandArgs.Command)
		}
		return nil
	},

	DefaultRouteHandler: executeHelp,

	// MattermostUserID is set for all commands, so no special "Requir" for it
	RouteHandlers: map[string]ActionScript{
		"connect":          commandConnect,
		"disconnect":       commandDisconnect,
		"settings":         commandSettings,
		"transition":       commandTransition,
		"install/server":   commandInstallServer,
		"install/cloud":    commandInstallCloud,
		"uninstall/cloud":  commandUninstall,
		"uninstall/server": commandUninstall,

		// used for debugging, uncomment if needed
		"webhook":         commandWebhookURL,
		"list":            commandList,
		"instance/select": commandInstanceSelect,
		"instance/delete": commandInstanceDelete,
	},
}

// Available settings
const (
	settingsNotifications = "notifications"
)

func (p *Plugin) ExecuteCommand(c *plugin.Context, commandArgs *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	args := strings.Fields(commandArgs.Command)
	args = args[1:]
	action := NewAction(p, c)
	action.CommandArgs = commandArgs
	action.Args = args

	_ = RequireMattermostUserId(action)

	scriptKey := ""
	for n := len(args); n > 0; n-- {
		key := strings.Join(args[:n], "/")
		if commandRouter.RouteHandlers[key] != nil {
			action.Args = args[n:]
			scriptKey = key
			break
		}
	}

	commandRouter.Run(scriptKey, action)
	return action.CommandResponse, nil
}

func executeHelp(a *Action) error {
	return a.RespondPrintf(helpText)
}

var commandConnect = ActionScript{
	RequireInstance,
	executeConnect,
}

func executeConnect(a *Action) error {
	if len(a.Args) != 0 {
		return executeHelp(a)
	}

	redirectURL, err := a.Instance.GetUserConnectURL(a.PluginConfig, a.SecretsStore, a.MattermostUserId)
	if err != nil {
		a.RespondError(0, err)
	}
	return a.RespondRedirect(redirectURL)
}

var commandDisconnect = ActionScript{
	RequireInstance,
	RequireJiraUser,
	executeDisconnect,
}

func executeDisconnect(a *Action) error {
	if len(a.Args) != 0 {
		return executeHelp(a)
	}

	err := DeleteUserInfoNotify(a.API, a.UserStore, a.Instance, a.MattermostUserId)
	if err != nil {
		return a.RespondError(0, err, "Could not complete the **disconnection** request")
	}

	return a.RespondPrintf("You have successfully disconnected your Jira account (**%s**).", a.JiraUser.Name)
}

const (
	settingOn  = "on"
	settingOff = "off"
)

var commandSettings = ActionScript{
	RequireJiraClient,
	executeSettings,
}

func executeSettings(a *Action) error {
	// TODO command-specific help
	// const helpText = "`/jira settings notifications [value]`\n* Invalid value. Accepted values are: `on` or `off`."
	if len(a.Args) != 2 {
		return executeHelp(a)
	}

	switch a.Args[0] {
	case settingsNotifications:
		value := false
		switch a.Args[1] {
		case settingOn:
			value = true
		case settingOff:
			value = false
		default:
			return executeHelp(a)
		}
		resp, err := UserSettingsNotifications(a.UserStore, a.Instance, a.MattermostUserId, a.JiraUser, value)
		if err != nil {
			return a.RespondError(0, err)
		}
		return a.RespondPrintf(resp)

	default:
		return a.RespondError(0, nil, "Unknown setting %q.", a.Args[0])
	}
}

var commandList = ActionScript{
	RequireMattermostSysAdmin,
	executeList,
}

func executeList(a *Action) error {
	if len(a.Args) != 0 {
		return executeHelp(a)
	}
	known, err := a.InstanceStore.LoadKnownInstances()
	if err != nil {
		return a.RespondError(0, err)
	}
	if len(known) == 0 {
		return a.RespondPrintf("(none installed)\n")
	}

	// error not important here, only need to highlight thee current in the list
	currentInstance, _ := a.CurrentInstanceStore.LoadCurrentInstance()

	keys := []string{}
	for key := range known {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	text := "Known Jira instances (selected instance is **bold**)\n\n| |URL|Type|\n|--|--|--|\n"
	for i, key := range keys {
		instance, err := a.InstanceStore.LoadInstance(key)
		if err != nil {
			text += fmt.Sprintf("|%v|%s|error: %v|\n", i+1, key, err)
			continue
		}
		details := ""
		for k, v := range instance.GetDisplayDetails() {
			details += fmt.Sprintf("%s:%s, ", k, v)
		}
		if len(details) > len(", ") {
			details = details[:len(details)-2]
		} else {
			details = instance.GetType()
		}
		format := "|%v|%s|%s|\n"
		if currentInstance != nil && key == currentInstance.GetURL() {
			format = "| **%v** | **%s** |%s|\n"
		}
		text += fmt.Sprintf(format, i+1, key, details)
	}
	return a.RespondPrintf(text)
}

var commandInstallCloud = ActionScript{
	RequireMattermostSysAdmin,
	executeInstallCloud,
}

func executeInstallCloud(a *Action) error {
	if len(a.Args) != 1 {
		return executeHelp(a)
	}
	jiraURL := a.Args[0]

	// Create an "uninitialized" instance of Jira Cloud that will
	// receive the /installed callback
	err := a.InstanceStore.CreateInactiveCloudInstance(jiraURL)
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
4. In the **From this URL field**, enter: %s%s
5. Wait for the app to install. Once completed, you should see an "Installed and ready to go!" message.
6. Use the "/jira connect" command to connect your Mattermost account with your Jira account.
7. Click the "More Actions" (...) option of any message in the channel (available when you hover over a message).

If you see an option to create a Jira issue, you're all set! If not, refer to our [documentation](https://about.mattermost.com/default-jira-plugin) for troubleshooting help.
`

	// TODO What is the exact group membership in Jira required? Site-admins?
	return a.RespondPrintf(addResponseFormat, jiraURL, jiraURL, a.PluginConfig.PluginURL, routeACJSON)
}

var commandInstallServer = ActionScript{
	RequireMattermostSysAdmin,
	executeInstallServer,
}

func executeInstallServer(a *Action) error {
	if len(a.Args) != 1 {
		return executeHelp(a)
	}
	jiraURL := a.Args[0]

	const addResponseFormat = `` +
		`Server instance has been installed. To finish the configuration, add an Application Link in your Jira instance following these steps:

1. Navigate to **Settings > Applications > Application Links**
2. Enter %s as the application link, then click **Create new link**.
3. In **Configure Application URL** screen, confirm your Mattermost URL is entered as the "New URL". Ignore any displayed errors and click **Continue**.
4. In **Link Applications** screen, set the following values:
  - **Application Name**: Mattermost
  - **Application Type**: Generic Application
5. Check the **Create incoming link** value, then click **Continue**.
6. In the following **Link Applications** screen, set the following values:
  - **Consumer Key**: %s
  - **Consumer Name**: Mattermost
  - **Public Key**: %s
7. Click **Continue**.
6. Use the "/jira connect" command to connect your Mattermost account with your Jira account.
7. Click the "More Actions" (...) option of any message in the channel (available when you hover over a message).

If you see an option to create a Jira issue, you're all set! If not, refer to our [documentation](https://about.mattermost.com/default-jira-plugin) for troubleshooting help.
`
	serverInstance := NewServerInstance(jiraURL, a.PluginConfig.PluginKey)
	err := a.InstanceStore.StoreInstance(serverInstance)
	if err != nil {
		return a.RespondError(0, err)
	}
	err = a.CurrentInstanceStore.StoreCurrentInstance(serverInstance)
	if err != nil {
		return a.RespondError(0, err)
	}

	pkey, err := publicKeyString(a.SecretsStore)
	if err != nil {
		return a.RespondError(0, err)
	}
	return a.RespondPrintf(addResponseFormat, a.PluginConfig.SiteURL, serverInstance.GetMattermostKey(), pkey)
}

var commandUninstall = ActionScript{
	RequireInstance,
	RequireMattermostSysAdmin,
	executeUninstall,
}

// executeUninstall will uninstall the jira cloud instance if the url matches, and then update all connected
// clients so that their Jira-related menu options are removed.
func executeUninstall(a *Action) error {
	if len(a.Args) != 1 {
		return executeHelp(a)
	}
	jiraURL := a.Args[0]

	if jiraURL != a.Instance.GetURL() {
		return a.RespondError(0, nil,
			"You have entered an incorrect URL. The current Jira instance URL is: %s. "+
				"Please enter the URL correctly to confirm the uninstall command.",
			a.Instance.GetURL())
	}

	err := a.InstanceStore.DeleteJiraInstance(a.Instance.GetURL())
	if err != nil {
		return a.RespondError(0, err,
			"Failed to delete Jira instance %s", a.Instance.GetURL())
	}

	// Notify users we have uninstalled an instance
	a.API.PublishWebSocketEvent(
		wSEventInstanceStatus,
		map[string]interface{}{
			"instance_installed": false,
		},
		&model.WebsocketBroadcast{},
	)

	const uninstallInstructions = `Jira instance successfully disconnected. Go to **Settings > Apps > Manage Apps** to remove the application in your Jira instance.`

	return a.RespondPrintf(uninstallInstructions)
}

var commandTransition = ActionScript{
	RequireJiraClient,
	executeTransition,
}

func executeTransition(a *Action) error {
	if len(a.Args) < 2 {
		return executeHelp(a)
	}
	issueKey := a.Args[0]
	toState := strings.Join(a.Args[1:], " ")

	msg, err := transitionJiraIssue(a, issueKey, toState)
	if err != nil {
		return a.RespondError(0, err)
	}
	return a.RespondPrintf(msg)
}

var commandWebhookURL = ActionScript{
	RequireMattermostSysAdmin,
	executeWebhookURL,
}

func executeWebhookURL(a *Action) error {
	if len(a.Args) != 0 {
		return executeHelp(a)
	}

	u, err := GetWebhookURL(a.PluginConfig, a.API, a.CommandArgs.TeamId, a.CommandArgs.ChannelId)
	if err != nil {
		return a.RespondError(0, err)
	}
	return a.RespondPrintf("Please use the following URL to set up a Jira webhook: %v", u)
}

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "jira",
		DisplayName:      "Jira",
		Description:      "Integration with Jira.",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: connect, disconnect, create, transition, settings, install cloud/server, uninstall cloud/server, help",
		AutoCompleteHint: "[command]",
	}
}

func commandResponsef(format string, args ...interface{}) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         fmt.Sprintf(format, args...),
		Username:     PluginMattermostUsername,
		IconURL:      PluginIconURL,
		Type:         model.POST_DEFAULT,
	}
}

var commandInstanceSelect = ActionScript{
	RequireMattermostSysAdmin,
	executeInstanceSelect,
}

func executeInstanceSelect(a *Action) error {
	if len(a.Args) != 1 {
		return executeHelp(a)
	}
	instanceKey := a.Args[0]
	num, err := strconv.ParseUint(instanceKey, 10, 8)
	if err == nil {
		known, loadErr := a.InstanceStore.LoadKnownInstances()
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
		instanceKey = keys[num-1]
	}

	instance, err := a.InstanceStore.LoadInstance(instanceKey)
	if err != nil {
		return a.RespondError(0, err)
	}
	err = a.CurrentInstanceStore.StoreCurrentInstance(instance)
	if err != nil {
		return a.RespondError(0, err)
	}

	a.Args = []string{}
	return executeList(a)
}

var commandInstanceDelete = ActionScript{
	RequireMattermostSysAdmin,
	executeInstanceDelete,
}

func executeInstanceDelete(a *Action) error {
	if len(a.Args) != 1 {
		return executeHelp(a)
	}
	instanceKey := a.Args[0]

	known, err := a.InstanceStore.LoadKnownInstances()
	if err != nil {
		return a.RespondError(0, err)
	}
	if len(known) == 0 {
		return a.RespondError(0, nil,
			"There are no instances to delete.\n")
	}

	num, err := strconv.ParseUint(instanceKey, 10, 8)
	if err == nil {
		if num < 1 || int(num) > len(known) {
			return a.RespondError(0, nil,
				"Wrong instance number %v, must be 1-%v\n", num, len(known)+1)
		}

		keys := []string{}
		for key := range known {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		instanceKey = keys[num-1]
	}

	// Remove the instance
	err = a.InstanceStore.DeleteJiraInstance(instanceKey)
	if err != nil {
		return a.RespondError(0, err)
	}

	// if that was our only instance, just respond with an empty list.
	if len(known) == 1 {
		a.Args = []string{}
		return executeList(a)
	}

	// Select instance #1
	a.Args = []string{"1"}
	return executeInstanceSelect(a)
}

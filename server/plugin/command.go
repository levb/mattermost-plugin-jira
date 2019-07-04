// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package plugin

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/command_action"
	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-plugin-jira/server/proxy"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
)

func commandConnect(a action.Action) error {
	ac := a.Context()
	redirectURL, err := ac.Upstream.GetUserConnectURL(ac.OneTimeStore, ac.PluginURL, ac.MattermostUserId)
	if err != nil {
		return a.RespondError(0, err, "command failed, please contact your system administrator")
	}
	return a.RespondRedirect(redirectURL)
}

func commandDisconnect(a action.Action) error {
	ac := a.Context()
	err := proxy.DeleteUserNotify(ac.API, ac.Upstream, ac.UpstreamUser)
	if err != nil {
		return a.RespondError(0, err, "Could not complete the **disconnection** request")
	}
	return a.RespondPrintf("You have successfully disconnected your Jira account (**%s**).",
		ac.UpstreamUser.UpstreamDisplayName())
}

const (
	settingOn  = "on"
	settingOff = "off"
)

func commandSettingsNotifications(a action.Action) error {
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
	err := jira.StoreUserSettingsNotifications(ac.Upstream, ac.UpstreamUser, value)
	if err != nil {
		return a.RespondError(0, err)
	}
	return a.RespondPrintf("Settings updated. Notifications %s.", valueStr)
}

func commandUpstreamList(a action.Action) error {
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

func commandUpstreamUninstall(a action.Action) error {
	ac := a.Context()
	upkey := ac.Upstream.Config().Key
	upstreamKey := a.FormValue("key")
	upstreamKey, err := upstream.NormalizeURL(upstreamKey)
	if err != nil {
		return a.RespondError(0, err)
	}

	if upstreamKey != upkey {
		return a.RespondError(0, nil,
			"You have entered an incorrect URL. The current Jira instance URL is: %s. "+
				"Please enter the URL correctly to confirm the uninstall command.",
			upkey)
	}

	err = proxy.DeleteUpstreamNotify(ac.API, ac.UpstreamStore, upstreamKey)
	if err != nil {
		return a.RespondError(0, err,
			"Failed to delete Jira instance %s", upstreamKey)
	}

	const uninstallInstructions = `` +
		`Jira instance successfully disconnected. Go to ` +
		`**Settings > Apps > Manage Apps** ` +
		`to remove the application in your Jira instance.`

	return a.RespondPrintf(uninstallInstructions)
}

func commandUpstreamSelect(a action.Action) error {
	ac := a.Context()
	upKey := a.FormValue("key")
	num, err := strconv.ParseUint(upKey, 10, 8)
	if err == nil {
		known, loadErr := ac.UpstreamStore.LoadKnown()
		if loadErr != nil {
			return a.RespondError(0, err,
				"Failed to load known upstreams")
		}
		if num < 1 || int(num) > len(known) {
			return a.RespondError(0, err,
				"Wrong upstream number %v, must be 1-%v\n", num, len(known)+1)
		}

		keys := []string{}
		for key := range known {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		upKey = keys[num-1]
	}

	upKey, err = upstream.NormalizeURL(upKey)
	if err != nil {
		return a.RespondError(0, err)
	}
	up, err := ac.UpstreamStore.Load(upKey)
	if err != nil {
		return a.RespondError(0, err,
			"Failed to load upstream %q", upKey)
	}
	err = proxy.StoreCurrentUpstreamNotify(ac.API, ac.UpstreamStore, up)
	if err != nil {
		return a.RespondError(0, err,
			"Failed to store current upstream")
	}

	a = command_action.CloneWithArgs(a, nil, nil)
	return commandUpstreamList(a)
}

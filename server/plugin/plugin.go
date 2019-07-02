// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package plugin

import (
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/command_action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/http_action"
	"github.com/mattermost/mattermost-plugin-jira/server/context"
	"github.com/mattermost/mattermost-plugin-jira/server/jira/jira_cloud"
	"github.com/mattermost/mattermost-plugin-jira/server/jira/jira_server"
	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

type Plugin struct {
	plugin.MattermostPlugin

	context.UpdateableContext

	Id      string
	Version string
}

var regexpNonAlnum = regexp.MustCompile("[^a-zA-Z0-9]+")

func (p *Plugin) OnActivate() error {
	kv := kvstore.NewPluginStore(p.API)
	bundlePath := filepath.Join(*(p.API.GetConfig().PluginSettings.Directory), p.Id)
	unmarshallers := map[string]upstream.Unmarshaller{
		jira_cloud.Type:  jira_cloud.Unmarshaller,
		jira_server.Type: jira_server.Unmarshaller,
	}

	f, err := MakeContext(p.API, kv, unmarshallers, p.Id, p.Version, bundlePath)
	if err != nil {
		return errors.WithMessage(err, "OnActivate failed")
	}
	p.UpdateContext(f)
	err = p.OnConfigurationChange()
	if err != nil {
		return errors.WithMessage(err, "OnActivate failed")
	}

	err = p.API.RegisterCommand(&model.Command{
		Trigger:          "jira",
		DisplayName:      "Jira",
		Description:      "Integration with Jira.",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: connect, disconnect, create, transition, settings, install cloud/server, uninstall cloud/server, help",
		AutoCompleteHint: "[command]",
	})
	if err != nil {
		p.API.LogError(err.Error())
		return errors.WithMessage(err, "OnActivate: failed to register command")
	}

	return nil
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	oldC := p.GetContext().Config
	newC := context.Config{}
	err := p.API.LoadPluginConfiguration(&newC)
	if err != nil {
		return errors.WithMessage(err, "failed to load plugin configuration")
	}
	// Load the public configuration fields from the Mattermost server configuration.
	mattermostSiteURL := *p.API.GetConfig().ServiceSettings.SiteURL

	newBotUserID := ""
	if newC.BotUserName != oldC.BotUserName {
		user, appErr := p.API.GetUserByUsername(newC.BotUserName)
		if appErr != nil {
			return errors.WithMessage(appErr, fmt.Sprintf("unable to load user %s", newC.BotUserName))
		}
		newBotUserID = user.Id
	}

	p.UpdateContext(func(c *context.Context) {
		RefreshContext(p.API, c, oldC, newC, mattermostSiteURL, newBotUserID)
	})
	return nil
}

func (p *Plugin) ServeHTTP(pc *plugin.Context, w http.ResponseWriter, r *http.Request) {
	// fmt.Println("<><> ", r.URL)
	a := action.NewAction(httpRouter, p.GetContext(), pc, "")
	a = http_action.Make(a, r, w)

	httpRouter.RunRoute(r.URL.Path, a)
}

func (p *Plugin) ExecuteCommand(pc *plugin.Context, commandArgs *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	a := action.NewAction(commandRouter, p.GetContext(), pc, "")
	key, a, err := command_action.Make(commandRouter, a, commandArgs)
	if err != nil {
		if a == nil {
			p.API.LogError(err.Error())
			return nil, model.NewAppError("Jira plugin", "", nil, err.Error(), 0)
		}
		a.RespondError(0, err, "command failed")
		return command_action.Response(a), nil
	}
	commandRouter.RunRoute(key, a)
	return command_action.Response(a), nil
}

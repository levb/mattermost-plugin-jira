// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package plugin

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/jira/jira_cloud"
	"github.com/mattermost/mattermost-plugin-jira/server/jira/jira_server"
	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/proxy"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

type Plugin struct {
	plugin.MattermostPlugin
	config SynchronizedConfig

	Id            string
	Version       string
	Templates     map[string]*template.Template
	Unmarshallers map[string]upstream.Unmarshaller

	proxy proxy.Proxy
}

func (p *Plugin) OnActivate() error {
	kv := kvstore.NewPluginStore(p.API)
	ots := kvstore.NewOneTimePluginStore(p.API, SessionTimeout)
	p.config.config = Config{
		kv:  kv,
		ots: ots,
	}

	// Tests override .Templates and .Unmarshallers so only load them if they
	// aren't already set
	if p.Unmarshallers == nil {
		p.Unmarshallers = map[string]upstream.Unmarshaller{
			jira_cloud.Type:  jira_cloud.Unmarshaller,
			jira_server.Type: jira_server.Unmarshaller,
		}
	}

	if len(p.Templates) == 0 {
		templatesPath := filepath.Join(*(p.API.GetConfig().PluginSettings.Directory),
			p.Id, "server", "dist", "templates")
		templates, err := loadTemplates(templatesPath)
		if err != nil {
			return err
		}
		p.Templates = templates
	}

	err := p.API.RegisterCommand(&model.Command{
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
	oldMain := p.config.Get().MainConfig
	newMain := MainConfig{}
	err := p.API.LoadPluginConfiguration(&newMain)
	if err != nil {
		return errors.WithMessage(err, "failed to load plugin configuration")
	}
	conf := p.config.Get()

	botUserId := conf.botUserId
	if newMain.BotUserName != oldMain.BotUserName {
		user, appErr := p.API.GetUserByUsername(newMain.BotUserName)
		if appErr != nil {
			return errors.WithMessage(appErr, fmt.Sprintf("unable to load user %s", newMain.BotUserName))
		}
		botUserId = user.Id
	}

	mattermostSiteURL := *p.API.GetConfig().ServiceSettings.SiteURL
	pluginKey := regexpNonAlnum.ReplaceAllString(strings.TrimRight(mattermostSiteURL, "/"), "_")
	pluginKey = "mattermost_" + regexpUnderlines.ReplaceAllString(pluginKey, "_")
	pluginURLPath := "/plugins/" + p.Id
	pluginURL := strings.TrimRight(mattermostSiteURL, "/") + pluginURLPath

	actionConfig := action.Config{
		API:                 p.API,
		BotIconURL:          "", // TODO where does it come from?
		BotUserId:           botUserId,
		BotUserName:         conf.MainConfig.BotUserName,
		EnableJiraUI:        conf.MainConfig.EnableJiraUI,
		PluginId:            p.Id,
		PluginKey:           pluginKey,
		PluginSiteURL:       mattermostSiteURL,
		PluginTemplates:     p.Templates,
		PluginURL:           pluginURL,
		PluginURLPath:       pluginURLPath,
		PluginVersion:       p.Version,
		PluginWebhookSecret: conf.MainConfig.WebhookSecret,
	}

	proxyConfig := proxy.Config{
		API:           p.API,
		KVStore:       conf.kv,
		Templates:     p.Templates,
		Unmarshallers: p.Unmarshallers,
	}

	newProxy, err := proxy.MakeProxy(proxyConfig, actionConfig)
	if err != nil {
		return err
	}

	p.config.Update(func(c *Config) {
		p.proxy = newProxy
		c.proxyConfig = proxyConfig
		c.actionConfig = actionConfig
	})

	return nil
}

func (p *Plugin) ServeHTTP(pc *plugin.Context, w http.ResponseWriter, r *http.Request) {
	_ = p.proxy.RunHTTP(pc, w, r)
}

func (p *Plugin) ExecuteCommand(pc *plugin.Context, commandArgs *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	resp, err := p.proxy.RunCommand(pc, commandArgs)
	if err != nil {
		p.API.LogError(err.Error())
		return nil, model.NewAppError("Jira plugin", "", nil, err.Error(), 0)
	}
	return resp, nil
}

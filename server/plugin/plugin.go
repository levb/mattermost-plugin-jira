// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package plugin

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"text/template"

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

var regexpNonAlnum = regexp.MustCompile("[^a-zA-Z0-9]+")
var regexpUnderlines = regexp.MustCompile("_+")

// MainConfig is the main plugin configuration, stored in the Mattermost config,
// and updated via Mattermost system console, CLI, or other means
type MainConfig struct {
	// Setting to turn on/off the webapp components of this plugin
	EnableJiraUI bool `json:"enablejiraui"`

	// Bot username
	BotUserName string `json:"username"`

	// Legacy 1.x Webhook secret
	WebhookSecret string `json:"secret"`
}

type Config struct {
	MainConfig
	proxyConfig proxy.Config
	actionConfig action.Config
}

type SynchronizedConfig struct {
	Config
	lock sync.RWMutex
}

func (c *SynchronizedConfig) GetConfig() Config {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.Config
}

func (c *SynchronizedConfig) UpdateContext(f func(*Config)) Config {
	c.lock.Lock()
	defer c.lock.Unlock()

	f(&c.Config)
	return c.Config
}

type Plugin struct {
	plugin.MattermostPlugin
	SynchronizedConfig
}

func (p *Plugin) OnActivate() error {
	kv := kvstore.NewPluginStore(p.API)
	unmarshallers := map[string]upstream.Unmarshaller{
		jira_cloud.Type:  jira_cloud.Unmarshaller,
		jira_server.Type: jira_server.Unmarshaller,
	}

	// Tests override .Templates
	if len(p.Templates) == 0 {
		templatesPath := filepath.Join(*(p.API.GetConfig().PluginSettings.Directory),
			p.Id, "server", "dist", "templates")
		templates, err := loadTemplates(templatesPath)
		if err != nil {
			return nil, err
		}
		p.Templates = templates
	}

	
	mattermostSiteURL := *api.GetConfig().ServiceSettings.SiteURL
	pluginKey := regexpNonAlnum.ReplaceAllString(strings.TrimRight(mattermostSiteURL, "/"), "_")
	pluginKey = "mattermost_" + regexpUnderlines.ReplaceAllString(pluginKey, "_")

	upstoreConfig := upstream.StoreConfig{
		RSAPrivateKey:   rsaPrivateKey,
		AuthTokenSecret: authTokenSecret,
		PluginKey:       pluginKey,
	}
		upstore := upstream.NewStore(api, upstoreConfig, kv, unmarshallers)
		if err != nil {
			return nil, err
		}
	
		// HW FUTURE TODO: Better template management, text vs html
		dir := filepath.Join(templatePath)
		dir := filepath.Join(bundlePath, "server", "dist", "templates")
		templates, err := loadTemplates(dir)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to load templates")
		}
	
		return func(c *context.Context) {
			c.StoreConfig = upstoreConfig
			c.MattermostSiteURL = mattermostSiteURL
			c.API = api
			c.UpstreamStore = upstore
			c.OneTimeStore = ots
			c.Templates = templates
			c.PluginId = pluginId
			c.PluginVersion = pluginVersion
			c.PluginURLPath = "/plugins/" + pluginId
			c.PluginURL = strings.TrimRight(c.MattermostSiteURL, "/") + c.PluginURLPath
		}, nil
	}
	
	


	f, err := MakeContext(p.API, kv, unmarshallers, p.Id, p.Version, templatesPath)
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

	newBotUserID := ""
	if newC.BotUserName != oldC.BotUserName {
		user, appErr := p.API.GetUserByUsername(newC.BotUserName)
		if appErr != nil {
			return errors.WithMessage(appErr, fmt.Sprintf("unable to load user %s", newC.BotUserName))
		}
		newBotUserID = user.Id
	}

	p.UpdateContext(func(c *context.Context) {
		RefreshContext(c, p.API, newC, newBotUserID)
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

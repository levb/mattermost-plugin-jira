// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/template"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/app/command"
	app_http "github.com/mattermost/mattermost-plugin-jira/server/app/http"
	"github.com/mattermost/mattermost-plugin-jira/server/instance"
	"github.com/mattermost/mattermost-plugin-jira/server/store"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	PluginMattermostUsername = "Jira Plugin"
	PluginIconURL            = "https://s3.amazonaws.com/mattermost-plugin-media/jira.jpg"
)

type storedConfig struct {
	// Bot username
	UserName string `json:"username"`

	// Legacy 1.x Webhook secret
	Secret string `json:"secret"`
}

type config struct {
	// StoredConfig caches values from the plugin's settings in the server's config.json
	storedConfig

	// Config Caches static values needed to make actions
	actionConfig action.Config
}

type Plugin struct {
	plugin.MattermostPlugin

	config
	confLock sync.RWMutex

	Id      string
	Version string
}

var regexpNonAlnum = regexp.MustCompile("[^a-zA-Z0-9]+")

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	oldSC := p.getConfig().storedConfig

	// Load the public configuration fields from the Mattermost server configuration.
	newSC := storedConfig{}
	err := p.API.LoadPluginConfiguration(&newSC)
	if err != nil {
		return errors.WithMessage(err, "failed to load plugin configuration")
	}

	newBotUserID := ""
	if newSC.UserName != oldSC.UserName {
		user, appErr := p.API.GetUserByUsername(newSC.UserName)
		if appErr != nil {
			return errors.WithMessage(appErr, fmt.Sprintf("unable to load user %s", newSC.UserName))
		}
		newBotUserID = user.Id
	}

	p.updateConfig(func(conf *config) {
		conf.storedConfig = newSC
		conf.actionConfig.BotUsername = newSC.UserName
		conf.actionConfig.WebhookSecret = newSC.Secret
		if newBotUserID != "" {
			conf.actionConfig.BotUserId = newBotUserID
		}
	})

	return nil
}

func (p *Plugin) OnActivate() error {
	s := store.NewPluginStore(p.API)
	instanceStore := instance.NewInstanceStore(s)

	dir := filepath.Join(*(p.API.GetConfig().PluginSettings.Directory), p.Id, "server", "dist", "templates")
	templates, err := p.loadTemplates(dir)
	if err != nil {
		return errors.WithMessage(err, "OnActivate: failed to load templates")
	}

	mattermostSiteURL := *p.API.GetConfig().ServiceSettings.SiteURL
	p.config = config{
		actionConfig: action.Config{
			API:          p.API,
			EnsuredStore: store.NewEnsuredStore(s),
			// UserStore is overwritten by RequireInstance
			UserStore:            store.NewUserStore(s),
			InstanceStore:        instanceStore,
			CurrentInstanceStore: instanceStore,

			// TODO text vs html templates
			Templates: templates,

			MattermostSiteURL: mattermostSiteURL,
			PluginId:          p.Id,
			PluginVersion:     p.Version,
			PluginKey:         "mattermost_" + regexpNonAlnum.ReplaceAllString(mattermostSiteURL, "_"),
			PluginURL:         strings.TrimRight(mattermostSiteURL, "/plugins/") + p.Id,
			PluginURLPath:     "/plugins/" + p.Id,

			// TODO
			BotIconURL: "",
		},
	}

	err = p.OnConfigurationChange()
	if err != nil {
		return errors.WithMessage(err, "OnActivate: failed to configure")
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
		return errors.WithMessage(err, "OnActivate: failed to register command")
	}

	return nil
}

func (p *Plugin) ServeHTTP(pc *plugin.Context, w http.ResponseWriter, r *http.Request) {
	a := action.MakeHTTPAction(app_http.Router, pc, p.getConfig().actionConfig, r, w)

	app_http.Router.RunRoute(r.URL.Path, a)
}

func (p *Plugin) ExecuteCommand(pc *plugin.Context, commandArgs *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	key, a, err := action.MakeCommandAction(command.Router, pc, p.getConfig().actionConfig, commandArgs)
	if err != nil {
		if a == nil {
			return nil, model.NewAppError("Jira plugin", "", nil, err.Error(), 0)
		}
		a.RespondError(0, err, "command failed")
		return a.CommandResponse, nil
	}
	command.Router.RunRoute(key, a)
	return a.CommandResponse, nil
}

func (p *Plugin) loadTemplates(dir string) (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		template, err := template.ParseFiles(path)
		if err != nil {
			p.API.LogError(fmt.Sprintf("OnActivate: failed to parse template %s: %v", path, err))
			return nil
		}
		key := path[len(dir):]
		templates[key] = template
		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "OnActivate: failed to load templates")
	}
	return templates, nil
}

func (p *Plugin) getConfig() config {
	p.confLock.RLock()
	defer p.confLock.RUnlock()
	return p.config
}

func (p *Plugin) updateConfig(f func(conf *config)) config {
	p.confLock.Lock()
	defer p.confLock.Unlock()

	f(&p.config)
	return p.config
}

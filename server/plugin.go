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
	"github.com/mattermost/mattermost-plugin-jira/server/app"
	"github.com/mattermost/mattermost-plugin-jira/server/app/command"
	app_http "github.com/mattermost/mattermost-plugin-jira/server/app/http"
	"github.com/mattermost/mattermost-plugin-jira/server/config"
	"github.com/mattermost/mattermost-plugin-jira/server/instance"
	"github.com/mattermost/mattermost-plugin-jira/server/instance/loader"
	"github.com/mattermost/mattermost-plugin-jira/server/store"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

type Plugin struct {
	plugin.MattermostPlugin

	config   config.Config
	confLock sync.RWMutex

	Id      string
	Version string
}

var regexpNonAlnum = regexp.MustCompile("[^a-zA-Z0-9]+")

func (p *Plugin) OnActivate() error {
	s := store.NewPluginStore(p.API)
	ots := store.NewPluginOneTimeStore(p.API, 60*15) // TTL 15 minutes
	instanceStore, currentInstanceStore, knownInstanceStore := instance.NewInstanceStore(s)

	rsaPrivateKey, err := app.EnsureRSAPrivateKey(s)
	if err != nil {
		return errors.WithMessage(err, "OnActivate failed")
	}
	authTokenSecret, err := app.EnsureAuthTokenSecret(s)
	if err != nil {
		return errors.WithMessage(err, "OnActivate failed")
	}
	instanceLoader := loader.New(instanceStore, currentInstanceStore, rsaPrivateKey, authTokenSecret)

	// HW FUTURE TODO: Better template management, text vs html
	dir := filepath.Join(*(p.API.GetConfig().PluginSettings.Directory), p.Id, "server", "dist", "templates")
	templates, err := p.loadTemplates(dir)
	if err != nil {
		return errors.WithMessage(err, "OnActivate: failed to load templates")
	}

	mattermostSiteURL := *p.API.GetConfig().ServiceSettings.SiteURL

	p.updateConfig(func(conf *config.Config) {
		conf.RSAPrivateKey = rsaPrivateKey
		conf.AuthTokenSecret = authTokenSecret

		conf.API = p.API
		conf.UserStore = store.NewUserStore(s)
		conf.InstanceStore = instanceStore
		conf.CurrentInstanceStore = currentInstanceStore
		conf.KnownInstancesStore = knownInstanceStore
		conf.InstanceLoader = instanceLoader
		conf.OneTimeStore = ots

		conf.Templates = templates
		conf.MattermostSiteURL = mattermostSiteURL
		conf.PluginId = p.Id
		conf.PluginVersion = p.Version
		conf.PluginKey = "mattermost_" + regexpNonAlnum.ReplaceAllString(mattermostSiteURL, "_")
		conf.PluginURL = strings.TrimRight(mattermostSiteURL, "/plugins/") + p.Id
		conf.PluginURLPath = "/plugins/" + p.Id
	})

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

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	oldSC := p.getConfig().StoredConfig

	// Load the public configuration fields from the Mattermost server configuration.
	newSC := config.StoredConfig{}
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

	p.updateConfig(func(conf *config.Config) {
		conf.StoredConfig = newSC
		if newBotUserID != "" {
			conf.BotUserId = newBotUserID
		}
	})

	return nil
}

func (p *Plugin) ServeHTTP(pc *plugin.Context, w http.ResponseWriter, r *http.Request) {
	a := action.MakeHTTPAction(app_http.Router, pc, p.getConfig(), r, w)

	app_http.Router.RunRoute(r.URL.Path, a)
}

func (p *Plugin) ExecuteCommand(pc *plugin.Context, commandArgs *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	key, a, err := action.MakeCommandAction(command.Router, pc, p.getConfig(), commandArgs)
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

func (p *Plugin) getConfig() config.Config {
	p.confLock.RLock()
	defer p.confLock.RUnlock()
	return p.config
}

func (p *Plugin) updateConfig(f func(conf *config.Config)) config.Config {
	p.confLock.Lock()
	defer p.confLock.Unlock()

	f(&p.config)
	return p.config
}

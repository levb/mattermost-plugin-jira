// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"fmt"
	gohttp "net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/template"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/command_action"
	"github.com/mattermost/mattermost-plugin-jira/server/action/http_action"
	"github.com/mattermost/mattermost-plugin-jira/server/config"
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

	context config.Context
	lock    sync.RWMutex

	Id      string
	Version string
}

var regexpNonAlnum = regexp.MustCompile("[^a-zA-Z0-9]+")

func (p *Plugin) OnActivate() error {
	s := kvstore.NewPluginStore(p.API)
	ots := kvstore.NewPluginOneTimeStore(p.API, 60*15) // TTL 15 minutes

	rsaPrivateKey, err := proxy.EnsureRSAPrivateKey(s)
	if err != nil {
		p.API.LogError(err.Error())
		return errors.WithMessage(err, "OnActivate failed")
	}
	authTokenSecret, err := proxy.EnsureAuthTokenSecret(s)
	if err != nil {
		p.API.LogError(err.Error())
		return errors.WithMessage(err, "OnActivate failed")
	}

	upstoreConfig := upstream.StoreConfig{
		RSAPrivateKey:   rsaPrivateKey,
		AuthTokenSecret: authTokenSecret,
	}

	upstore := upstream.NewStore(upstoreConfig, s,
		map[string]upstream.Unmarshaller{
			jira_cloud.Type:  jira_cloud.Unmarshaller,
			jira_server.Type: jira_server.Unmarshaller,
		})

	// HW FUTURE TODO: Better template management, text vs html
	dir := filepath.Join(*(p.API.GetConfig().PluginSettings.Directory), p.Id, "server", "dist", "templates")
	templates, err := p.loadTemplates(dir)
	if err != nil {
		p.API.LogError(err.Error())
		return errors.WithMessage(err, "OnActivate: failed to load templates")
	}

	p.updateContext(func(c *config.Context) {
		c.API = p.API
		c.UpstreamStore = upstore
		c.OneTimeStore = ots

		c.Templates = templates
		c.PluginId = p.Id
		c.PluginVersion = p.Version
		c.PluginURLPath = "/plugins/" + p.Id
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
		p.API.LogError(err.Error())
		return errors.WithMessage(err, "OnActivate: failed to register command")
	}

	return nil
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	oldC := p.getContext().Config

	// Load the public configuration fields from the Mattermost server configuration.
	newC := config.Config{}
	err := p.API.LoadPluginConfiguration(&newC)
	if err != nil {
		err = errors.WithMessage(err, "failed to load plugin configuration")
		p.API.LogError(err.Error())
		return err
	}

	mattermostSiteURL := *p.API.GetConfig().ServiceSettings.SiteURL

	newBotUserID := ""
	if newC.BotUserName != oldC.BotUserName {
		user, appErr := p.API.GetUserByUsername(newC.BotUserName)
		if appErr != nil {
			p.API.LogError(appErr.Error())
			return errors.WithMessage(appErr, fmt.Sprintf("unable to load user %s", newC.BotUserName))
		}
		newBotUserID = user.Id
	}

	p.updateContext(func(c *config.Context) {
		c.Config = newC
		c.MattermostSiteURL = mattermostSiteURL
		c.PluginKey = "mattermost_" + regexpNonAlnum.ReplaceAllString(c.MattermostSiteURL, "_")
		c.PluginURLPath = "/plugins/" + manifest.Id
		c.PluginURL = strings.TrimRight(c.MattermostSiteURL, "/") + c.PluginURLPath

		if newBotUserID != "" {
			c.BotUserId = newBotUserID
		}
	})

	return nil
}

func (p *Plugin) ServeHTTP(pc *plugin.Context, w gohttp.ResponseWriter, r *gohttp.Request) {
	fmt.Println("<><> ", r.URL)
	a := action.NewAction(httpRouter, p.getContext(), pc, "")
	a = http_action.Make(a, r, w)

	httpRouter.RunRoute(r.URL.Path, a)
}

func (p *Plugin) ExecuteCommand(pc *plugin.Context, commandArgs *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	a := action.NewAction(commandRouter, p.getContext(), pc, "")
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

func (p *Plugin) getContext() config.Context {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.context
}

func (p *Plugin) updateContext(f func(conf *config.Context)) config.Context {
	p.lock.Lock()
	defer p.lock.Unlock()

	f(&p.context)
	return p.context
}

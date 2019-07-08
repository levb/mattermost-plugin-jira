// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package plugin

import (
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"text/template"
	"time"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/proxy"
)

const SessionTimeout = 15. * time.Minute

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
	proxyConfig  proxy.Config
	actionConfig action.Config

	KVStore      kvstore.KVStore
	OneTimeStore kvstore.OneTimeStore
	BotIconURL   string
	BotUserId    string
}

type SynchronizedConfig struct {
	*Config
	lock sync.RWMutex
}

func (c *SynchronizedConfig) Get() Config {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return *c.Config
}

func (c *SynchronizedConfig) Update(f func(*Config)) Config {
	c.lock.Lock()
	defer c.lock.Unlock()

	f(c.Config)
	return *c.Config
}

func loadTemplates(dir string) (map[string]*template.Template, error) {
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

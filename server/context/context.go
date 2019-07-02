// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package context

import (
	"crypto/rsa"
	"sync"
	"text/template"

	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/plugin"
)

// Config is the main plugin configuration, stored in the Mattermost config,
// and updated via Mattermost system console, CLI, or other means
type Config struct {
	// Setting to turn on/off the webapp components of this plugin
	EnableJiraUI bool `json:"enablejiraui"`

	// Bot username
	BotUserName string `json:"username"`

	// Legacy 1.x Webhook secret
	WebhookSecret string `json:"secret"`
}

// Context is the run-time execution context
type Context struct {
	Config

	// Plugin-wide secrets
	RSAPrivateKey   *rsa.PrivateKey
	AuthTokenSecret []byte

	// Service dependencies
	API           plugin.API
	UpstreamStore upstream.Store
	OneTimeStore  kvstore.OneTimeStore

	// Parsed and cached templates
	// TODO store text vs html templates
	Templates map[string]*template.Template

	MattermostSiteURL string
	PluginId          string
	PluginVersion     string
	PluginKey         string
	PluginURL         string
	PluginURLPath     string

	// BotUserID caches the bot user ID (derived from the configured UserName)
	BotUserId  string
	BotIconURL string
}

type UpdateableContext struct {
	context Context
	lock    sync.RWMutex
}

func (c *UpdateableContext) GetContext() Context {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.context
}

func (c *UpdateableContext) UpdateContext(f func(conf *Context)) Context {
	c.lock.Lock()
	defer c.lock.Unlock()

	f(&c.context)
	return c.context
}

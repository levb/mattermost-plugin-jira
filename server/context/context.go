// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package context

import (
	"sync"
	"text/template"

	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/plugin"
)

// Context is the run-time execution context
type Context struct {
	Config
	upstream.StoreConfig

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
	PluginURL         string
	PluginURLPath     string

	// BotUserID caches the bot user ID (derived from the configured UserName)
	BotUserId  string
	BotIconURL string
}

type SynchronizedContext struct {
	context Context
	lock    sync.RWMutex
}

func (c *SynchronizedContext) GetContext() Context {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.context
}

func (c *SynchronizedContext) UpdateContext(f func(conf *Context)) Context {
	c.lock.Lock()
	defer c.lock.Unlock()

	f(&c.context)
	return c.context
}
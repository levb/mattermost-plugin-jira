// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package config

import (
	"crypto/rsa"
	"text/template"

	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-plugin-jira/server/store"
	mmplugin "github.com/mattermost/mattermost-server/plugin"
)

type StoredConfig struct {
	// Bot username
	UserName string `json:"username"`

	// Legacy 1.x Webhook secret
	WebhookSecret string `json:"secret"`
}

type Config struct {
	StoredConfig

	// Plugin-wide secrets
	RSAPrivateKey   *rsa.PrivateKey
	AuthTokenSecret []byte

	// Service dependencies
	API                  mmplugin.API
	UserStore            upstream.UserStore
	UpstreamStore        upstream.UpstreamStore
	KnownUpstreamsStore  upstream.KnownUpstreamsStore
	CurrentUpstreamStore upstream.CurrentUpstreamStore
	OneTimeStore         store.OneTimeStore

	Templates map[string]*template.Template // TODO text vs html templates

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

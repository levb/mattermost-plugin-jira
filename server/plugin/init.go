// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/context"
	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/proxy"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/plugin"
)

func MakeContext(api plugin.API, kv kvstore.KVStore, unmarshallers map[string]upstream.Unmarshaller,
	pluginId, pluginVersion, bundlePath string) (func(*context.Context), error) {

	ots := kvstore.NewPluginOneTimeStore(api, 60*15) // TTL 15 minutes

	rsaPrivateKey, err := proxy.EnsureRSAPrivateKey(kv)
	if err != nil {
		return nil, err
	}
	authTokenSecret, err := proxy.EnsureAuthTokenSecret(kv)
	if err != nil {
		return nil, err
	}
	upstoreConfig := upstream.StoreConfig{
		RSAPrivateKey:   rsaPrivateKey,
		AuthTokenSecret: authTokenSecret,
	}
	upstore := upstream.NewStore(upstoreConfig, kv, unmarshallers)
	if err != nil {
		return nil, err
	}

	// HW FUTURE TODO: Better template management, text vs html
	dir := filepath.Join(bundlePath, "server", "dist", "templates")
	templates, err := loadTemplates(dir)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load templates")
	}

	return func(c *context.Context) {
		c.API = api
		c.UpstreamStore = upstore
		c.OneTimeStore = ots
		c.Templates = templates
		c.PluginId = pluginId
		c.PluginVersion = pluginVersion
		c.PluginURLPath = "/plugins/" + pluginId
	}, nil
}

func RefreshContext(api plugin.API, c *context.Context, oldC, newC context.Config, mattermostSiteURL, newBotUserID string) {
	c.Config = newC
	c.MattermostSiteURL = mattermostSiteURL
	c.PluginKey = "mattermost_" + regexpNonAlnum.ReplaceAllString(c.MattermostSiteURL, "_")
	c.PluginURLPath = "/plugins/" + c.PluginId
	c.PluginURL = strings.TrimRight(c.MattermostSiteURL, "/") + c.PluginURLPath

	if newBotUserID != "" {
		c.BotUserId = newBotUserID
	}
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

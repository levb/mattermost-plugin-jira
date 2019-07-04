// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package plugin

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/context"
	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/proxy"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/plugin"
)

var regexpNonAlnum = regexp.MustCompile("[^a-zA-Z0-9]+")
var regexpUnderlines = regexp.MustCompile("_+")

func MakeContext(api plugin.API, kv kvstore.KVStore, unmarshallers map[string]upstream.Unmarshaller,
	pluginId, pluginVersion, templatePath string) (func(*context.Context), error) {

	ots := kvstore.NewOneTimePluginStore(api, 15*time.Minute)

	rsaPrivateKey, err := proxy.EnsureRSAPrivateKey(kv)
	if err != nil {
		return nil, err
	}
	authTokenSecret, err := proxy.EnsureAuthTokenSecret(kv)
	if err != nil {
		return nil, err
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

func RefreshContext(c *context.Context, api plugin.API, newC context.Config, newBotUserID string) {
	c.Config = newC
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

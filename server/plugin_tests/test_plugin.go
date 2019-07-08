// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package plugin_tests

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"

	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/plugin"
)

var defaultMMConfig = &model.Config{
	ServiceSettings: model.ServiceSettings{
		SiteURL: model.NewString("http://localhost:8065"),
	},
}

func SetupTestPlugin(t *testing.T, api *plugintest.API, conf plugin.Config,
	mmconfig *model.Config) *plugin.Plugin {

	if mmconfig == nil {
		mmconfig = defaultMMConfig
	}

	// For now, use the mocked store for OTS as is, there is no expiry
	kv := kvstore.NewMockedStore()
	p := &plugin.Plugin{
		Templates: map[string]*template.Template{},
		Config: plugin.SynchronizedConfig{
			Config: &plugin.Config{
				KVStore:      kv,
				OneTimeStore: kvstore.NewOneTimeStore(kv),
			},
		},
	}

	api.On("GetUserByUsername", mock.AnythingOfTypeArgument("string")).Return(&model.User{}, nil)
	api.On("LogDebug",
		mock.AnythingOfTypeArgument("string")).Return(nil)
	api.On("LogInfo",
		mock.AnythingOfTypeArgument("string")).Return(nil)
	api.On("RegisterCommand",
		mock.AnythingOfTypeArgument("*model.Command")).Return(nil)
	api.On("GetConfig").Return(mmconfig)
	api.On("LoadPluginConfiguration",
		mock.AnythingOfTypeArgument(
			"*plugin.MainConfig")).Return(
		func(cc interface{}) error {
			c := cc.(*plugin.MainConfig)
			*c = conf.MainConfig
			return nil
		})

	p.SetAPI(api)

	//	error api.LoadPluginConfiguration(interface{})

	err := p.OnActivate()
	require.Nil(t, err)

	err = p.OnConfigurationChange()
	require.Nil(t, err)

	return p
}

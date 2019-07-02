// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package teststore

import (
	"encoding/json"
	"testing"

	"github.com/mattermost/mattermost-plugin-jira/server/context"
	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/plugin"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin/plugintest"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	UpstreamA_Type     = "type-a"
	UpstreamA_Key      = "upstream1"
	UpstreamA_URL      = "https://mmtest.madeup-1.notreal"
	UpstreamB_Type     = "type-b"
	UpstreamB_Key      = "https://mmtest.madeup-2.notreal"
	UpstreamB_URL      = "https://mmtest.madeup-2.notreal"
	UserA_MattermostId = "mmuser__1_1234567890123456"
	UserA_UpstreamId   = "up1_user_1"
	UserB_MattermostId = "mmuser__2_1234567890123456"
	UserB_UpstreamId   = "up2_user_2"
	KeyDoesNotExist    = "-"
)

type UpstreamA struct {
	upstream.BasicUpstream
	A string
	C string
}

type UserA struct {
	upstream.BasicUser
	A string
}

type UpstreamB struct {
	upstream.BasicUpstream
	B string
	C string
}

type UserB struct {
	upstream.BasicUser
	B string
}

type unmarshallerA struct{}

func (_ unmarshallerA) UnmarshalUser(data []byte, mattermostUserId string) (upstream.User, error) {
	u := UserA{}
	err := json.Unmarshal(data, &u)
	if err != nil {
		return nil, err
	}
	u.MUserId = mattermostUserId
	return &u, nil
}

func (_ unmarshallerA) UnmarshalUpstream(data []byte, basicUp upstream.BasicUpstream) (upstream.Upstream, error) {
	up := UpstreamA{}
	err := json.Unmarshal(data, &up)
	if err != nil {
		return nil, err
	}
	up.BasicUpstream = basicUp
	return &up, nil
}

type unmarshallerB struct{}

func (_ unmarshallerB) UnmarshalUser(data []byte, mattermostUserId string) (upstream.User, error) {
	u := UserB{}
	err := json.Unmarshal(data, &u)
	if err != nil {
		return nil, err
	}
	u.MUserId = mattermostUserId
	return &u, nil
}

func (_ unmarshallerB) UnmarshalUpstream(data []byte, basicUp upstream.BasicUpstream) (upstream.Upstream, error) {
	up := UpstreamB{}
	err := json.Unmarshal(data, &up)
	if err != nil {
		return nil, err
	}
	up.BasicUpstream = basicUp
	return &up, nil
}

var Unmarshallers = map[string]upstream.Unmarshaller{
	UpstreamA_Type: &unmarshallerA{},
	UpstreamB_Type: &unmarshallerB{},
}

func UpstreamStore_2Upstreams2Users(t *testing.T, s upstream.Store) {
	up1 := &UpstreamA{
		BasicUpstream: s.MakeBasicUpstream(
			upstream.UpstreamConfig{
				StoreConfig: *s.Config(),
				Type:        UpstreamA_Type,
				Key:         UpstreamA_Key,
				URL:         UpstreamA_URL,
			}),
		A: "aaa",
		C: "bbb",
	}

	up2 := &UpstreamB{
		BasicUpstream: s.MakeBasicUpstream(
			upstream.UpstreamConfig{
				StoreConfig: *s.Config(),
				Type:        UpstreamB_Type,
				Key:         UpstreamB_Key,
				URL:         UpstreamB_URL,
			}),
		B: "aaa-aaa",
		C: "ccc",
	}

	err := s.Store(up1)
	require.Nil(t, err)
	err = s.Store(up2)
	require.Nil(t, err)
	err = s.StoreCurrent(up2)
	require.Nil(t, err)

	user1 := upstream.NewBasicUser(UserA_MattermostId, UserA_UpstreamId)
	err = up1.StoreUser(user1)
	require.Nil(t, err)

	user2 := upstream.NewBasicUser(UserB_MattermostId, UserB_UpstreamId)
	err = up2.StoreUser(user2)
	require.Nil(t, err)
}

func SetupTestPlugin(t *testing.T, api *plugintest.API, conf context.Config) *plugin.Plugin {
	kv := kvstore.NewMockedStore()
	p := &plugin.Plugin{}
	p.SetAPI(api)

	api.On("GetUserByUsername", mock.AnythingOfTypeArgument("string")).Return(&model.User{}, nil)
	api.On("LogDebug",
		mock.AnythingOfTypeArgument("string")).Return(nil)
	api.On("LogInfo",
		mock.AnythingOfTypeArgument("string")).Return(nil)

	f, err := plugin.MakeContext(p.API, kv, Unmarshallers, "pluginID", "version-string", "..")
	require.Nil(t, err)
	p.UpdateContext(f)
	p.UpdateContext(func(c *context.Context) {
		plugin.RefreshContext(p.API, c, context.Config{}, conf, "site.url", "")
	})

	return p
}

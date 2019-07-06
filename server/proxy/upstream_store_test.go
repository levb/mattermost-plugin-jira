// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package proxy

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/action"
	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
)

const (
	up1Type           = "type1"
	up1Key            = "upstream1"
	up1URL            = "https://mmtest.madeup-1.notreal"
	up2Type           = "type2"
	up2Key            = "https://mmtest.madeup-2.notreal"
	up2URL            = "https://mmtest.madeup-2.notreal"
	user1MattermostId = "mmuser__1_1234567890123456"
	user1UpstreamId   = "up1_user_1"
	user2MattermostId = "mmuser__2_1234567890123456"
	user2UpstreamId   = "up2_user_2"
)

type testUpstream1 struct {
	upstream.Basic
	A string
	B string
}

type testUser1 struct {
	upstream.BasicUser
	A string
}

func (up testUpstream1) LoadUser(mattermostUserId string) (upstream.User, error) {
	data, err := up.LoadUserRaw(mattermostUserId)
	if err != nil {
		return nil, err
	}

	user := testUser1{}
	err = json.Unmarshal(data, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

type testUpstream2 struct {
	upstream.Basic
	A string
	C string
}

type testUser2 struct {
	upstream.BasicUser
	B string
}

func (up testUpstream2) LoadUser(mattermostUserId string) (upstream.User, error) {
	data, err := up.LoadUserRaw(mattermostUserId)
	if err != nil {
		return nil, err
	}

	user := testUser2{}
	err = json.Unmarshal(data, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

type unmarshaller1 struct{}

func (_ unmarshaller1) UnmarshalUpstream(data []byte, basicUp upstream.Basic) (upstream.Upstream, error) {
	up := testUpstream1{}
	err := json.Unmarshal(data, &up)
	if err != nil {
		return nil, err
	}
	up.Basic = basicUp
	return &up, nil
}

type unmarshaller2 struct{}

func (_ unmarshaller2) UnmarshalUpstream(data []byte, basicUp upstream.Basic) (upstream.Upstream, error) {
	up := testUpstream2{}
	err := json.Unmarshal(data, &up)
	if err != nil {
		return nil, err
	}
	up.Basic = basicUp
	return &up, nil
}

func setupProxyWith2(t *testing.T, kv kvstore.KVStore) (Proxy, upstream.Upstream, upstream.Upstream) {
	conf := Config{
		KVStore: kv,
		Unmarshallers: map[string]upstream.Unmarshaller{
			up1Type: unmarshaller1{},
			up2Type: unmarshaller2{},
		},
	}
	aconf := action.Config{}
	p, err := MakeProxy(conf, aconf)
	require.Nil(t, err)

	up1 := &testUpstream1{
		Basic: p.MakeBasicUpstream(up1Key, up1Type),
		A:     "aaa",
		B:     "bbb",
	}

	up2 := &testUpstream2{
		Basic: p.MakeBasicUpstream(up2Key, up2Type),
		A:     "aaa-aaa",
		C:     "ccc",
	}

	return p, up1, up2
}

func TestUpstreamStore(t *testing.T) {
	mockedStore := kvstore.NewMockedStore()
	p, up1, up2 := setupProxyWith2(t, mockedStore)

	t.Run("1-Initial not found", func(t *testing.T) {
		_, err := p.LoadUpstream(up1Key)
		require.Equal(t, err, kvstore.ErrNotFound)
		err = p.DeleteUpstream(up1Key)
		require.Equal(t, err, kvstore.ErrNotFound)
		_, err = p.LoadKnownUpstreams()
		require.Equal(t, err, kvstore.ErrNotFound)
		_, err = p.LoadCurrentUpstream()
		require.Equal(t, err, kvstore.ErrNotFound)
		_, err = p.LoadCurrentUpstreamRaw()
		require.Equal(t, err, kvstore.ErrNotFound)
	})

	t.Run("2-Store", func(t *testing.T) {
		err := p.StoreUpstream(up1)
		require.NoError(t, err)
		err = p.StoreUpstream(up2)
		require.NoError(t, err)
		err = p.StoreCurrentUpstream(up2)
		require.NoError(t, err)
	})

	t.Run("3-Check KV", func(t *testing.T) {
		require.Equal(t, 6, len(mockedStore.Values))
		require.NotNil(t, mockedStore.Values[kvstore.KeyRSAPrivateKey])
		require.NotNil(t, mockedStore.Values[kvstore.KeyTokenSecret])
		require.NotNil(t, mockedStore.Values[kvstore.KeyCurrentUpstream])
		require.NotNil(t, mockedStore.Values[kvstore.KeyKnownUpstreams])
		require.NotNil(t, mockedStore.Values[kvstore.KeyPrefixUpstream+"3fcdd98b4aad6829424697769125c9a0"])
		require.NotNil(t, mockedStore.Values[kvstore.KeyPrefixUpstream+"4a8477d5dd6a9175599aad82ba0a3261"])
	})

	t.Run("4-Load", func(t *testing.T) {
		up, err := p.LoadUpstream(up1Key)
		require.NoError(t, err)
		require.Equal(t, up1, up)
		up, err = p.LoadUpstream(up2Key)
		require.NoError(t, err)
		require.Equal(t, up2, up)
		up, err = p.LoadCurrentUpstream()
		require.NoError(t, err)
		require.Equal(t, up2, up)
		known, err := p.LoadKnownUpstreams()
		require.NoError(t, err)
		require.Equal(t, 2, len(known))
		require.Equal(t, up1Type, known[up1Key])
		require.Equal(t, up2Type, known[up2Key])
		data, err := p.LoadCurrentUpstreamRaw()
		require.NoError(t, err)
		require.Equal(t, `{"Key":"https://mmtest.madeup-2.notreal","Type":"type2","A":"aaa-aaa","C":"ccc"}`, string(data))
	})

	t.Run("5-Check KV", func(t *testing.T) {
		require.Equal(t, 6, len(mockedStore.Values))
		require.NotNil(t, mockedStore.Values[kvstore.KeyRSAPrivateKey])
		require.NotNil(t, mockedStore.Values[kvstore.KeyTokenSecret])
		require.NotNil(t, mockedStore.Values[kvstore.KeyCurrentUpstream])
		require.NotNil(t, mockedStore.Values[kvstore.KeyKnownUpstreams])
		require.NotNil(t, mockedStore.Values[kvstore.KeyPrefixUpstream+"3fcdd98b4aad6829424697769125c9a0"])
		require.NotNil(t, mockedStore.Values[kvstore.KeyPrefixUpstream+"4a8477d5dd6a9175599aad82ba0a3261"])
	})

	t.Run("6-Delete 2", func(t *testing.T) {
		err := p.DeleteUpstream(up2Key)
		require.NoError(t, err)
	})

	t.Run("7-Verify after delete 2", func(t *testing.T) {
		require.Equal(t, 4, len(mockedStore.Values))
		require.NotNil(t, mockedStore.Values[kvstore.KeyRSAPrivateKey])
		require.NotNil(t, mockedStore.Values[kvstore.KeyTokenSecret])
		require.NotNil(t, mockedStore.Values[kvstore.KeyKnownUpstreams])
		require.NotNil(t, mockedStore.Values[kvstore.KeyPrefixUpstream+"3fcdd98b4aad6829424697769125c9a0"])

		known, err := p.LoadKnownUpstreams()
		require.NoError(t, err)
		require.Equal(t, 1, len(known))
		require.Equal(t, "type1", known[up1Key])
	})

	t.Run("8-Delete 1", func(t *testing.T) {
		err := p.DeleteUpstream(up1Key)
		require.NoError(t, err)
	})

	t.Run("9-Verify after delete 1", func(t *testing.T) {
		require.Equal(t, 3, len(mockedStore.Values))
		require.NotNil(t, mockedStore.Values[kvstore.KeyRSAPrivateKey])
		require.NotNil(t, mockedStore.Values[kvstore.KeyTokenSecret])
		require.NotNil(t, mockedStore.Values[kvstore.KeyKnownUpstreams])

		known, err := p.LoadKnownUpstreams()
		require.NoError(t, err)
		require.Equal(t, 0, len(known))
	})
}

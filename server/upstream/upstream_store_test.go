// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package upstream

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
)

type testUpstream1 struct {
	BasicUpstream
	A string
	B string
}

type testUser1 struct {
	User
	A string
}

type testUpstream2 struct {
	BasicUpstream
	A string
	C string
}

type testUser2 struct {
	User
	B string
}

func unmarshalUser(data []byte, userRef interface{}) (User, error) {
	err := json.Unmarshal(data, &userRef)
	if err != nil {
		return nil, err
	}
	return userRef.(User), nil
}

func unmarshalUpstream(data []byte, storeConf StoreConfig, upstreamRef interface{}) (Upstream, error) {
	err := json.Unmarshal(data, &upstreamRef)
	if err != nil {
		return nil, err
	}
	up := upstreamRef.(Upstream)
	up.Config().StoreConfig = storeConf
	return up, nil
}

type unmarshaller1 struct{}

func (_ unmarshaller1) UnmarshalUser(data []byte) (User, error) {
	u := testUser1{}
	return unmarshalUser(data, &u)
}

func (_ unmarshaller1) UnmarshalUpstream(data []byte, storeConf StoreConfig) (Upstream, error) {
	up := testUpstream1{}
	return unmarshalUpstream(data, storeConf, &up)
}

type unmarshaller2 struct{}

func (_ unmarshaller2) UnmarshalUser(data []byte) (User, error) {
	u := testUser2{}
	return unmarshalUser(data, &u)
}

func (_ unmarshaller2) UnmarshalUpstream(data []byte, storeConf StoreConfig) (Upstream, error) {
	up := testUpstream2{}
	return unmarshalUpstream(data, storeConf, &up)
}

func TestMakeBasicUpstream(t *testing.T) {
	s := NewStore(
		StoreConfig{
			AuthTokenSecret: []byte("secret"),
		},
		kvstore.NewMockedStore(), nil)

	config := UpstreamConfig{
		StoreConfig: *s.Config(),
		Type:        "type1",
		Key:         "key1",
		URL:         "URL1",
	}
	up := s.MakeBasicUpstream(config)
	require.Equal(t, config, *up.Config())
}
func TestUpstreamStore(t *testing.T) {
	mockedStore := kvstore.NewMockedStore()

	s := NewStore(StoreConfig{},
		&mockedStore,
		map[string]Unmarshaller{
			"type1": unmarshaller1{},
			"type2": unmarshaller2{},
		},
	)

	up1Key := "upstream1"
	up1URL := "https://mmtest.madeup-1.notreal"
	up2Key := "https://mmtest.madeup-2.notreal"
	up2URL := "https://mmtest.madeup-2.notreal"

	up1 := testUpstream1{
		BasicUpstream: s.MakeBasicUpstream(
			UpstreamConfig{
				StoreConfig: *s.Config(),
				Type:        "type1",
				Key:         up1Key,
				URL:         up1URL,
			}),
		A: "aaa",
		B: "bbb",
	}

	up2 := testUpstream2{
		BasicUpstream: s.MakeBasicUpstream(
			UpstreamConfig{
				StoreConfig: *s.Config(),
				Type:        "type2",
				Key:         up2Key,
				URL:         up2URL,
			}),
		A: "aaa-aaa",
		C: "ccc",
	}

	t.Run("Initial not found", func(t *testing.T) {
		_, err := s.Load(up1Key)
		require.Equal(t, err, kvstore.ErrNotFound)
		err = s.Delete(up1Key)
		require.Equal(t, err, kvstore.ErrNotFound)
		_, err = s.LoadKnown()
		require.Equal(t, err, kvstore.ErrNotFound)
		_, err = s.LoadCurrent()
		require.Equal(t, err, kvstore.ErrNotFound)
		_, err = s.LoadCurrentRaw()
		require.Equal(t, err, kvstore.ErrNotFound)
	})

	t.Run("Store", func(t *testing.T) {
		err := s.Store(&up1)
		require.NoError(t, err)
		err = s.Store(&up2)
		require.NoError(t, err)
		err = s.StoreCurrent(&up2)
		require.NoError(t, err)
	})

	t.Run("Check KV", func(t *testing.T) {
		require.Equal(t, 4, len(mockedStore.Values))
		require.NotNil(t, mockedStore.Values[keyCurrentUpstream])
		require.NotNil(t, mockedStore.Values[keyKnownUpstreams])
		require.NotNil(t, mockedStore.Values[prefixUpstream+"3fcdd98b4aad6829424697769125c9a0"])
		require.NotNil(t, mockedStore.Values[prefixUpstream+"4a8477d5dd6a9175599aad82ba0a3261"])
	})

	t.Run("Load", func(t *testing.T) {
		up, err := s.Load(up1Key)
		require.NoError(t, err)
		require.Equal(t, up1.Config(), up.Config())
		up, err = s.Load(up2Key)
		require.NoError(t, err)
		require.Equal(t, up2.Config(), up.Config())
		up, err = s.LoadCurrent()
		require.NoError(t, err)
		require.Equal(t, up2.Config(), up.Config())
		known, err := s.LoadKnown()
		require.NoError(t, err)
		require.Equal(t, 2, len(known))
		require.Equal(t, "type1", known[up1Key])
		require.Equal(t, "type2", known[up2Key])
		data, err := s.LoadCurrentRaw()
		require.NoError(t, err)
		require.Equal(t, `{"Key":"https://mmtest.madeup-2.notreal","URL":"https://mmtest.madeup-2.notreal","Type":"type2","A":"aaa-aaa","C":"ccc"}`, string(data))
	})

	t.Run("Check KV", func(t *testing.T) {
		require.Equal(t, 4, len(mockedStore.Values))
		require.NotNil(t, mockedStore.Values[keyCurrentUpstream])
		require.NotNil(t, mockedStore.Values[keyKnownUpstreams])
		require.NotNil(t, mockedStore.Values[prefixUpstream+"3fcdd98b4aad6829424697769125c9a0"])
		require.NotNil(t, mockedStore.Values[prefixUpstream+"4a8477d5dd6a9175599aad82ba0a3261"])
	})

	t.Run("Delete 2", func(t *testing.T) {
		err := s.Delete(up2Key)
		require.NoError(t, err)
	})

	t.Run("Verify after delete 2", func(t *testing.T) {
		require.Equal(t, 2, len(mockedStore.Values))
		require.NotNil(t, mockedStore.Values[keyKnownUpstreams])
		require.NotNil(t, mockedStore.Values[prefixUpstream+"3fcdd98b4aad6829424697769125c9a0"])

		known, err := s.LoadKnown()
		require.NoError(t, err)
		require.Equal(t, 1, len(known))
		require.Equal(t, "type1", known[up1Key])
	})

	t.Run("Delete 1", func(t *testing.T) {
		err := s.Delete(up1Key)
		require.NoError(t, err)
	})

	t.Run("Verify after delete 1", func(t *testing.T) {
		require.Equal(t, 1, len(mockedStore.Values))
		require.NotNil(t, mockedStore.Values[keyKnownUpstreams])

		known, err := s.LoadKnown()
		require.NoError(t, err)
		require.Equal(t, 0, len(known))
	})
}

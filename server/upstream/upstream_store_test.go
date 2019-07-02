// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package upstream

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
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
	BasicUpstream
	A string
	B string
}

type testUser1 struct {
	BasicUser
	A string
}

type testUpstream2 struct {
	BasicUpstream
	A string
	C string
}

type testUser2 struct {
	BasicUser
	B string
}

type unmarshaller1 struct{}

func (_ unmarshaller1) UnmarshalUser(data []byte, mattermostUserId string) (User, error) {
	u := testUser1{}
	err := json.Unmarshal(data, &u)
	if err != nil {
		return nil, err
	}
	u.MUserId = mattermostUserId
	return &u, nil
}

func (_ unmarshaller1) UnmarshalUpstream(data []byte, basicUp BasicUpstream) (Upstream, error) {
	up := testUpstream1{}
	err := json.Unmarshal(data, &up)
	if err != nil {
		return nil, err
	}
	up.BasicUpstream = basicUp
	return &up, nil
}

type unmarshaller2 struct{}

func (_ unmarshaller2) UnmarshalUser(data []byte, mattermostUserId string) (User, error) {
	u := testUser2{}
	err := json.Unmarshal(data, &u)
	if err != nil {
		return nil, err
	}
	u.MUserId = mattermostUserId
	return &u, nil
}

func (_ unmarshaller2) UnmarshalUpstream(data []byte, basicUp BasicUpstream) (Upstream, error) {
	up := testUpstream2{}
	err := json.Unmarshal(data, &up)
	if err != nil {
		return nil, err
	}
	up.BasicUpstream = basicUp
	return &up, nil
}

func setupUpstreamStoreWith2(underlying kvstore.KVStore) (Store, Upstream, Upstream) {
	s := NewStore(StoreConfig{},
		underlying,
		map[string]Unmarshaller{
			up1Type: unmarshaller1{},
			up2Type: unmarshaller2{},
		},
	)

	up1 := &testUpstream1{
		BasicUpstream: s.MakeBasicUpstream(
			UpstreamConfig{
				StoreConfig: *s.Config(),
				Type:        up1Type,
				Key:         up1Key,
				URL:         up1URL,
			}),
		A: "aaa",
		B: "bbb",
	}

	up2 := &testUpstream2{
		BasicUpstream: s.MakeBasicUpstream(
			UpstreamConfig{
				StoreConfig: *s.Config(),
				Type:        up2Type,
				Key:         up2Key,
				URL:         up2URL,
			}),
		A: "aaa-aaa",
		C: "ccc",
	}

	return s, up1, up2
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
	s, up1, up2 := setupUpstreamStoreWith2(mockedStore)

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
		err := s.Store(up1)
		require.NoError(t, err)
		err = s.Store(up2)
		require.NoError(t, err)
		err = s.StoreCurrent(up2)
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
		require.Equal(t, up1Type, known[up1Key])
		require.Equal(t, up2Type, known[up2Key])
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

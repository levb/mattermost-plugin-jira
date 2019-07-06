// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package proxy

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/kvstore"
	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
)

func TestUserStore(t *testing.T) {
	mockedStore := kvstore.NewMockedStore()
	_, up1, up2 := setupProxyWith2(t, mockedStore)

	t.Run("1-Initial not found", func(t *testing.T) {
		_, err := up1.LoadUser(user1MattermostId)
		require.Equal(t, err, kvstore.ErrNotFound)
		_, err = up2.LoadUser(user2MattermostId)
		require.Equal(t, err, kvstore.ErrNotFound)
	})

	user1 := testUser1{
		BasicUser: *upstream.NewBasicUser(user1MattermostId, user1UpstreamId),
		A:         "aaa",
	}
	user2 := testUser2{
		BasicUser: *upstream.NewBasicUser(user2MattermostId, user2UpstreamId),
		B:         "bbb",
	}

	t.Run("2-Store", func(t *testing.T) {
		err := up1.StoreUser(&user1)
		require.NoError(t, err)
		err = up2.StoreUser(&user2)
		require.NoError(t, err)
	})

	t.Run("3-Check KV", func(t *testing.T) {
		require.Equal(t, 6, len(mockedStore.Values))
		require.NotNil(t, mockedStore.Values[kvstore.KeyRSAPrivateKey])
		require.NotNil(t, mockedStore.Values[kvstore.KeyTokenSecret])
		require.NotNil(t, mockedStore.Values["a118119fea2366ef738746f733dee9ad"])
		require.NotNil(t, mockedStore.Values["b66b968eb080c0b0d6628d906570ee12"])
		require.NotNil(t, mockedStore.Values["c363a1f460eec519cb338177e3fe12dd"])
		require.NotNil(t, mockedStore.Values["ddd9e4290f3a6eabe07720398c81e2c4"])
	})

	t.Run("4-Load", func(t *testing.T) {
		u, err := up1.LoadUser(user1MattermostId)
		require.NoError(t, err)
		uu1, ok := u.(*testUser1)
		require.True(t, ok)
		require.Equal(t, *uu1, user1)

		u, err = up2.LoadUser(user2MattermostId)
		require.NoError(t, err)
		uu2, ok := u.(*testUser2)
		require.True(t, ok)
		require.Equal(t, *uu2, user2)
	})

	t.Run("5-Not found on the wrong upstream", func(t *testing.T) {
		_, err := up2.LoadUser(user1MattermostId)
		require.Equal(t, err, kvstore.ErrNotFound)
		_, err = up1.LoadUser(user2MattermostId)
		require.Equal(t, err, kvstore.ErrNotFound)
	})

	t.Run("6-Delete 2", func(t *testing.T) {
		err := up2.DeleteUser(&user2)
		require.NoError(t, err)
	})

	t.Run("7-Verify after delete 2", func(t *testing.T) {
		require.Equal(t, 4, len(mockedStore.Values))
		require.NotNil(t, mockedStore.Values[kvstore.KeyRSAPrivateKey])
		require.NotNil(t, mockedStore.Values[kvstore.KeyTokenSecret])
		require.NotNil(t, mockedStore.Values["c363a1f460eec519cb338177e3fe12dd"])
		require.NotNil(t, mockedStore.Values["ddd9e4290f3a6eabe07720398c81e2c4"])
		_, err := up2.LoadUser(user2MattermostId)
		require.Equal(t, err, kvstore.ErrNotFound)
	})

	t.Run("8-Delete 1", func(t *testing.T) {
		err := up1.DeleteUser(&user1)
		require.NoError(t, err)
	})

	t.Run("9-Verify after delete 1", func(t *testing.T) {
		require.Equal(t, 2, len(mockedStore.Values))
		require.NotNil(t, mockedStore.Values[kvstore.KeyRSAPrivateKey])
		require.NotNil(t, mockedStore.Values[kvstore.KeyTokenSecret])
	})
}

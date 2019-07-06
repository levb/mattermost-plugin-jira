// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package kvstore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPluginStore(t *testing.T) {
	testStore(t, "plugin-", NewMockedStore())
}

func testStore(t *testing.T, name string, s KVStore) {
	t.Run(name+"Load-Store-Delete", func(t *testing.T) {
		loaded, err := s.Load("key")
		require.Equal(t, err, ErrNotFound)

		err = s.Store("key", []byte("somedata"))
		require.NoError(t, err)
		loaded, err = s.Load("key")
		require.NoError(t, err)
		require.Equal(t, []byte("somedata"), loaded)

		err = s.Store("key", []byte("newdata"))
		require.NoError(t, err)
		loaded, err = s.Load("key")
		require.NoError(t, err)
		require.Equal(t, []byte("newdata"), loaded)

		err = s.Delete("key")
		require.NoError(t, err)
		loaded, err = s.Load("key")
		require.Equal(t, err, ErrNotFound)
	})

	t.Run(name+"Ensure", func(t *testing.T) {
		value, err := Ensure(s, "ensured key", []byte("first value"))
		require.NoError(t, err)
		require.Equal(t, []byte("first value"), value)
		value, err = Ensure(s, "ensured key", []byte("second value"))
		require.NoError(t, err)
		require.Equal(t, []byte("first value"), value)
	})
}

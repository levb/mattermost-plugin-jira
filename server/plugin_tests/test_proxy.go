// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package plugin_tests

import (
	"testing"

	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/stretchr/testify/require"
)

func Store2Upstreams2Users(t *testing.T, upstore upstream.UpstreamStore) {
	up1 := &UpstreamA{
		Basic: upstore.MakeBasicUpstream(UpstreamA_Key, UpstreamA_Type),
		A:     "aaa",
		C:     "bbb",
	}

	up2 := &UpstreamB{
		Basic: upstore.MakeBasicUpstream(UpstreamB_Key, UpstreamB_Type),
		B:     "aaa-aaa",
		C:     "ccc",
	}

	err := upstore.StoreUpstream(up1)
	require.Nil(t, err)
	err = upstore.StoreUpstream(up2)
	require.Nil(t, err)
	err = upstore.StoreCurrentUpstream(up2)
	require.Nil(t, err)

	user1 := upstream.NewBasicUser(UserA_MattermostId, UserA_UpstreamId)
	err = up1.StoreUser(user1)
	require.Nil(t, err)

	user2 := upstream.NewBasicUser(UserB_MattermostId, UserB_UpstreamId)
	err = up2.StoreUser(user2)
	require.Nil(t, err)
}

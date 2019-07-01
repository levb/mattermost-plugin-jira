// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira_cloud

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewAuthToken(t *testing.T) {
	up := Upstream{}
	up.BasicUpstream.UpstreamConfig.StoreConfig.AuthTokenSecret = []byte("abcdefghijABCDEFGHIJabcdefghijXY")

	mmtoken, err := up.newAuthToken("01234567890123456789012345", "secret_0")
	require.Nil(t, err)

	id, secret, err := up.parseAuthToken(mmtoken)
	require.Nil(t, err)
	require.Equal(t, "01234567890123456789012345", id)
	require.Equal(t, "secret_0", secret)
}

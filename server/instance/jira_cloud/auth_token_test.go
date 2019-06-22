// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jira_cloud

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstance_ParseAuthToken(t *testing.T) {
	tests := []struct {
		name             string
		mattermostUserId string
		authTokenSecret  []byte
		secret           []byte
		encoded          string
		wantErr          bool
	}{
		// TODO: Add test cases.
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cloudInstance := Instance{
				authTokenSecret: tc.authTokenSecret,
			}

			token, err := cloudInstance.NewAuthToken(tc.mattermostUserId, tc.authTokenSecret)

			got, got1, err := cloudInstance.ParseAuthToken(tt.args.encoded)
			require.Equal(t, tt.wantErr, (err != nil))
			if got != tt.want {
				t.Errorf("Instance.ParseAuthToken() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Instance.ParseAuthToken() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

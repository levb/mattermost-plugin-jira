package main

import (
	"testing"

	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest/mock"
	"github.com/stretchr/testify/require"
)

func TestMigrateV2Instances(t *testing.T) {
	tests := map[string]struct {
		known                string
		current              string
		expectInstances      string
		expectInstance       string
		numExpectedInstances int
	}{
		"Server": {
			known: `{"http://localhost:8080":"server"}`,
			current: `{
				"Key":"http://localhost:8080",
				"Type":"server",
				"PluginVersion":"2.4.0",
				"JIRAServerURL":"http://localhost:8080",
				"MattermostKey":"mattermost_https_levb_ngrok_io"
			}`,
			expectInstance:       `{"PluginVersion":"3.0.0","InstanceID":"http://localhost:8080","Type":"server","IsV2Legacy":true,"MattermostKey":"mattermost_https_levb_ngrok_io","JIRAServerURL":"http://localhost:8080"}`,
			expectInstances:      `[{"PluginVersion":"3.0.0","InstanceID":"http://localhost:8080","Type":"server","IsV2Legacy":true}]`,
			numExpectedInstances: 1,
		},
		"Cloud": {
			known: `{"https://mmtest.atlassian.net":"cloud"}`,
			current: `{
				"Key": "https://mmtest.atlassian.net",
				"Type": "cloud",
				"PluginVersion": "2.4.0",
				"Installed": true,
				"RawAtlassianSecurityContext": "{\"BaseURL\":\"https://mmtest.atlassian.net\"}"
			}`,
			expectInstance:       `{"PluginVersion":"3.0.0","InstanceID":"https://mmtest.atlassian.net","Type":"cloud","IsV2Legacy":true,"Installed":true,"RawAtlassianSecurityContext":"{\"BaseURL\":\"https://mmtest.atlassian.net\"}"}`,
			expectInstances:      `[{"PluginVersion":"3.0.0","InstanceID":"https://mmtest.atlassian.net","Type":"cloud","IsV2Legacy":true}]`,
			numExpectedInstances: 1,
		},
		"No Instance Installed": {
			known:                `{}`,
			current:              "",
			numExpectedInstances: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api := &plugintest.API{}

			api.On("LogError", mock.AnythingOfTypeArgument("string")).Return(nil)
			api.On("LogDebug", mock.AnythingOfTypeArgument("string")).Return(nil)

			api.On("KVGet", keyInstances).Return(nil, nil)
			api.On("KVGet", v2keyKnownJiraInstances).Return([]byte(tc.known), nil)
			if tc.current != "" {
				api.On("KVGet", v2keyCurrentJIRAInstance).Return([]byte(tc.current), nil)
			} else {
				api.On("KVGet", v2keyCurrentJIRAInstance).Return(nil, nil)
			}

			storedInstancePayload := []byte{}
			storedInstancesPayload := []byte{}
			api.On("KVSet", mock.AnythingOfTypeArgument("string"), mock.AnythingOfTypeArgument("[]uint8")).Return(nil).Run(
				func(args mock.Arguments) {
					key := args.Get(0).(string)
					switch key {
					case "jira_instance_b5f8e96862ed24709919a73271ae8851",
						"jira_instance_37d007a56d816107ce5b52c10342db37":
						storedInstancePayload = args.Get(1).([]byte)
					case "instances/v3":
						storedInstancesPayload = args.Get(1).([]byte)
					default:
						t.Fatalf("Unexpected key in KVSet: %q", key)
					}
				})

			p := &Plugin{}
			p.SetAPI(api)
			store := NewStore(p)
			p.instanceStore = store
			manifest.Version = "3.0.0"

			instances, err := MigrateV2Instances(p)
			require.NoError(t, err)

			require.Equal(t, tc.numExpectedInstances, instances.Len())
			if instances.Len() > 0 {
				id := instances.IDs()[0]
				require.Equal(t, tc.expectInstance, string(storedInstancePayload))
				require.Equal(t, tc.expectInstances, string(storedInstancesPayload))
				require.Equal(t, id, instances.Get(id).GetID())
			}
		})
	}
}

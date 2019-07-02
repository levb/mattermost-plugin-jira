// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"github.com/mattermost/mattermost-plugin-jira/server/plugin"
	server_plugin "github.com/mattermost/mattermost-server/plugin"
)

func main() {
	server_plugin.ClientMain(&plugin.Plugin{
		Id:      manifest.Id,
		Version: manifest.Version,
	})
}

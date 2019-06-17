// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	"github.com/mattermost/mattermost-plugin-jira/server/plugin"
	mmplugin "github.com/mattermost/mattermost-server/plugin"
)

func main() {
	mmplugin.ClientMain(&plugin.Plugin{
		Id:      manifest.Id,
		Version: manifest.Version,
	})
}

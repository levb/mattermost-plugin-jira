// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package main

import (
	mmplugin "github.com/mattermost/mattermost-server/plugin"
)

func main() {
	mmplugin.ClientMain(&Plugin{
		Id:      manifest.Id,
		Version: manifest.Version,
	})
}

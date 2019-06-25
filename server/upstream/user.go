// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package upstream

type User interface {
	Key() string
	DisplayName() string

	// Returns a writeable pointer. Clone or lock to use in goroutines
	Settings() *UserSettings
}

type UserSettings struct {
	Notifications bool `json:"notifications"`
}

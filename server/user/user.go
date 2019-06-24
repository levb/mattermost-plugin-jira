// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package user

type User interface {
	Key() string
	DisplayName() string

	// Returns a writeable pointer. Clone or lock to use in goroutines
	Settings() *Settings
}

type Settings struct {
	Notifications bool `json:"notifications"`
}

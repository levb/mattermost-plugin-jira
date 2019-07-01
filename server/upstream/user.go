// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package upstream

type User interface {
	MattermostUserId() string
	MattermostDisplayName() string
	UpstreamUserId() string
	UpstreamDisplayName() string
	Settings() *UserSettings
}

type UserSettings struct {
	Notifications bool `json:"notifications"`
}

type BasicUser struct {
	MUserId      string       `json:"mattermost_user_id"`
	UUserId      string       `json:"upstream_user_id"`
	UserSettings UserSettings `json:"settings"`
}

func (u BasicUser) MattermostUserId() string      { return u.MUserId }
func (u BasicUser) MattermostDisplayName() string { return "" }
func (u BasicUser) UpstreamUserId() string        { return u.UUserId }
func (u BasicUser) UpstreamDisplayName() string   { return "" }
func (u *BasicUser) Settings() *UserSettings      { return &u.UserSettings }

var DefaultUserSettings = UserSettings{
	Notifications: true,
}

func NewBasicUser(mattermostUserId, upstreamUserId string) BasicUser {
	return BasicUser{
		MUserId:      mattermostUserId,
		UUserId:      upstreamUserId,
		UserSettings: DefaultUserSettings,
	}
}

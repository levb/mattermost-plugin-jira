// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package upstream

type User interface {
	MattermostId() string
	MattermostDisplayName() string
	UpstreamId() string
	UpstreamDisplayName() string
	Settings() *UserSettings
}

type UserSettings struct {
	Notifications bool `json:"notifications"`
}

type BasicUser struct {
	MattermostUserId string       `json:"mattermost_user_id"`
	UpstreamUserId   string       `json:"upstream_user_id"`
	UserSettings     UserSettings `json:"settings"`
}

func (u BasicUser) MattermostId() string          { return u.MattermostUserId }
func (u BasicUser) MattermostDisplayName() string { return "" }
func (u BasicUser) UpstreamId() string            { return u.UpstreamUserId }
func (u BasicUser) UpstreamDisplayName() string   { return "" }
func (u *BasicUser) Settings() *UserSettings      { return &u.UserSettings }

var DefaultUserSettings = UserSettings{
	Notifications: true,
}

func NewBasicUser(mattermostUserId, upstreamUserId string) BasicUser {
	return BasicUser{
		MattermostUserId: mattermostUserId,
		UpstreamUserId:   upstreamUserId,
		UserSettings:     DefaultUserSettings,
	}
}

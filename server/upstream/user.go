// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package upstream

type User interface {
	MattermostId() string
	MattermostDisplayName() string
	UpstreamId() string
	UpstreamDisplayName() string

	// Settings returns a writeable pointer. Clone or lock to use in goroutines.
	Settings() *UserSettings
}

type UserSettings struct {
	Notifications bool `json:"notifications"`
}

type user struct {
	MattermostUserId string       `json:"mattermost_user_id"`
	UpstreamUserId   string       `json:"upstream_user_id"`
	UserSettings     UserSettings `json:"settings"`
}

func (u user) MattermostId() string          { return u.MattermostUserId }
func (u user) UpstreamId() string            { return u.UpstreamUserId }
func (u user) MattermostDisplayName() string { return "" }
func (u user) UpstreamDisplayName() string   { return "" }
func (u user) Settings() *UserSettings       { return &u.UserSettings }

func NewUser(mattermostUserId, upstreamUserId string) User {
	return &user{
		MattermostUserId: mattermostUserId,
		UpstreamUserId:   upstreamUserId,
	}
}

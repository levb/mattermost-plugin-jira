// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package plugin_tests

import (
	"encoding/json"

	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
)

const (
	UpstreamA_Type     = "type-a"
	UpstreamA_Key      = "upstream1"
	UpstreamA_URL      = "https://mmtest.madeup-1.notreal"
	UpstreamB_Type     = "type-b"
	UpstreamB_Key      = "https://mmtest.madeup-2.notreal"
	UpstreamB_URL      = "https://mmtest.madeup-2.notreal"
	UserA_MattermostId = "mmuser__1_1234567890123456"
	UserA_UpstreamId   = "up1_user_1"
	UserB_MattermostId = "mmuser__2_1234567890123456"
	UserB_UpstreamId   = "up2_user_2"
	KeyDoesNotExist    = "-"
)

type UpstreamA struct {
	upstream.Basic
	A string
	C string
}

type UserA struct {
	upstream.BasicUser
	A string
}

type UpstreamB struct {
	upstream.Basic
	B string
	C string
}

type UserB struct {
	upstream.BasicUser
	B string
}

type unmarshallerA struct{}

func (_ unmarshallerA) UnmarshalUpstream(data []byte, basicUp upstream.Basic) (upstream.Upstream, error) {
	up := UpstreamA{}
	err := json.Unmarshal(data, &up)
	if err != nil {
		return nil, err
	}
	up.Basic = basicUp
	return &up, nil
}

type unmarshallerB struct{}

func (_ unmarshallerB) UnmarshalUpstream(data []byte, basicUp upstream.Basic) (upstream.Upstream, error) {
	up := UpstreamB{}
	err := json.Unmarshal(data, &up)
	if err != nil {
		return nil, err
	}
	up.Basic = basicUp
	return &up, nil
}

var Unmarshallers = map[string]upstream.Unmarshaller{
	UpstreamA_Type: &unmarshallerA{},
	UpstreamB_Type: &unmarshallerB{},
}

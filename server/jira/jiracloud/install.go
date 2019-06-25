// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package jiracloud

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/mattermost/mattermost-plugin-jira/server/upstream"
	"github.com/mattermost/mattermost-plugin-jira/server/store"
	"github.com/mattermost/mattermost-plugin-jira/server/jira"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
)

func ProcessInstalled(
	api plugin.API,
	s store.Store,
	upstreamStore upstream.UpstreamStore,
	currentUpstreamStore upstream.CurrentUpstreamStore,
	authTokenSecret []byte,
	mattermostKey string,
	body io.Reader) (int, error) {

	data, err := ioutil.ReadAll(body)
	if err != nil {
		return http.StatusBadRequest, errors.WithMessage(err, "failed to read request")
	}

	var asc AtlassianSecurityContext
	err = json.Unmarshal(data, &asc)
	if err != nil {
		return http.StatusBadRequest, errors.WithMessage(err, "failed to unmarshal request")
	}

	// Only allow this operation once, a Jira upstream must already exist
	// for asc.BaseURL but not Installed.
	up := Upstream{}
	err = upstreamStore.Load(asc.BaseURL, &up)
	if err == store.ErrNotFound {
		return http.StatusNotFound,
			errors.Errorf("Jira upstream %q must first be added to Mattermost", asc.BaseURL)
	}
	if err != nil {
		return http.StatusInternalServerError,
			errors.WithMessagef(err, "failed to load Jira upstream %q", asc.BaseURL)
	}
	if up.Installed {
		return http.StatusForbidden,
			errors.Errorf("Jira upstream %q is already installed", asc.BaseURL)
	}

	upstream := upstream.NewUpstream(upstream.UpstreamConfig{
		AuthTokenSecret: authTokenSecret,
		Key: asc.BaseURL, 
		URL: asc.BaseURL, 
		Type: Type,
		MattermostKey:  mattermostKey,
	}, s)

	up = NewUpstream(upstream, true, string(data), &asc, authTokenSecret)

	// UpstreamStore.Store also updates the list of known Jira upstreams
	err = upstreamStore.Store(up)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	err = jira.StoreCurrentUpstreamAndNotify(api, currentUpstreamStore, cloudUpstream)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

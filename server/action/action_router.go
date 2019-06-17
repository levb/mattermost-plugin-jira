// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package action

import (
	"strings"
)

type Router struct {
	FindHandler    func(route string) Func
	DefaultHandler Func
	LogHandler     Func
}

func (ar Router) RunRoute(route string, a Action) {
	route = strings.TrimRight(route, "/")

	// See if we have a handler for the exact route match
	handler := ar.FindHandler(route)
	if handler == nil {
		// Look for a subpath match
		handler = ar.FindHandler(route + "/*")
	}

	// Look for a /* above
	for handler == nil {
		n := strings.LastIndex(route, "/")
		if n == -1 {
			break
		}
		handler = ar.FindHandler(route[:n] + "/*")
		route = route[:n]
	}

	// Use the default, if needed
	if handler == nil {
		handler = ar.DefaultHandler
	}

	// Run the handler
	err := handler(a)
	if err != nil {
		return
	}

	// Log
	if ar.LogHandler != nil {
		_ = ar.LogHandler(a)
	}
}

type Script []Func

func (script Script) Run(a Action) error {
	for _, f := range script {
		if f == nil {
			continue
		}
		err := f(a)
		if err != nil {
			a.Context().LogErr = err
			return err
		}
	}
	return nil
}

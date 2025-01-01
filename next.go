// Copyright (c) Liam Stanley <liam@liam.sh>. All rights reserved. Use of
// this source code is governed by the MIT license that can be found in
// the LICENSE file.

package pt

import (
	"net/http"
	"net/url"
	"strings"
)

// NextKey is the query param name used when redirecting.
var NextKey = "next"

// This can be used within templates like..:
// 	<a href="/auth/login{% if url.Path != '/' %}?next={{ url.Path|urlencode }}{% endif %}</a>
//
// Or maybe:
// 	<input type="hidden" name="next" value="{{ url.Query().Get('next') }}">

// GetNextURL obtains the target url from the intermediary URL, allowing you
// to pass in a URL parameter (e.g. ?next=/some/authed/page) which you can
// redirect to after doing some task (e.g. authenticating).
//
// For example:
//
//	if isAuthed(r) {
//		if next := pt.GetNextURL(r); next != "" {
//			pt.RedirectToNextURL(w, r, http.StatusFound)
//			return
//		}
//
//		http.Redirect(w, r, "/some/home/page", http.StatusFound)
//		return
//	}
func GetNextURL(r *http.Request) (next string) {
	_ = r.ParseForm()

	if next = r.URL.Query().Get(NextKey); next == "" {
		next = r.FormValue(NextKey)
	}

	if next == "" {
		return next
	}

	if qnext, err := url.QueryUnescape(next); err == nil {
		if !strings.HasPrefix(qnext, "/") {
			return ""
		}

		return qnext
	}

	if !strings.HasPrefix(next, "/") {
		return ""
	}

	return next
}

// RedirectWithNextURL redirects to another page and passes the next url param,
// (e.g. a login page). target is the target redirect page, and statusCode is
// the http code used when redirecting.
//
// Example:
//
//	if !auth(user, passwd) {
//		pt.RedirectWithNextURL(w, r, r.URL.EscapedPath(), http.StatusTemporaryRedirect)
//		return
//	}
func RedirectWithNextURL(w http.ResponseWriter, r *http.Request, target string, statusCode int) {
	http.Redirect(w, r, target+"?"+NextKey+"="+url.QueryEscape(GetNextURL(r)), statusCode)
}

// RedirectToNextURL redirects to the url specified within the "next" query
// parameter. Do this after the task (e.g. after authenticating).
//
// Example:
//
//	if auth(user, passwd) {
//		pt.RedirectToNextURL(w, r, http.StatusTemporaryRedirect)
//		return
//	}
func RedirectToNextURL(w http.ResponseWriter, r *http.Request, statusCode int) {
	http.Redirect(w, r, GetNextURL(r), statusCode)
}

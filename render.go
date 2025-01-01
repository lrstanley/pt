// Copyright (c) Liam Stanley <liam@liam.sh>. All rights reserved. Use of
// this source code is governed by the MIT license that can be found in
// the LICENSE file.

package pt

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/flosch/pongo2/v6"
)

// M is a convenience alias for quickly building a map structure that is going
// out to a responder. Just a short-hand.
type M map[string]interface{}

// New returns a new loader with initialized template sets and configuration.
func New(set string, conf Config) *Loader {
	if conf.Loader == nil && conf.FS == nil {
		panic("no loader provided")
	}

	if conf.ErrorLogger == nil {
		conf.ErrorLogger = io.Discard
	}

	var fileServer pongo2.TemplateLoader
	if conf.Loader != nil {
		fileServer = &memLoader{loaderFunc: conf.Loader}
	} else {
		fileServer = pongo2.NewFSLoader(conf.FS)
	}

	ld := &Loader{
		fs: pongo2.NewSet(set, fileServer),
		ts: time.Now(), conf: &conf,
	}

	return ld
}

// Config is the configuration which should be passed to New().
type Config struct {
	// CacheParsed caches the parsed version of the template in memory,
	// which is useful for production when templates aren't being loaded
	// while the application is running (or when you are using ricebox or
	// similar.)
	CacheParsed bool
	// Loader is the template loader to use to load a template. This can
	// be some kind of filesystem loader, or a assetfs/memory-based loader
	// (re: go-ricebox).
	//
	// For example:
	//   rice.MustFindBox("static").Bytes
	Loader func(path string) ([]byte, error)
	FS     fs.FS
	// DefaultCtx is an optional function which you can supply, which is
	// called every time the Render() function is called, which allows you
	// to add additional context variables to the ctx map. Useful if you are
	// adding variables to multiple handlers frequently.
	DefaultCtx func(http.ResponseWriter, *http.Request) (ctx map[string]interface{})
	// NotFoundHandler is an optional handler which you can define when the
	// template cannot be found based on what's returned from the Loader
	// method. If this is not defined, the Render() function will panic, as
	// this indicates the use of an undefined template.
	NotFoundHandler http.HandlerFunc
	// ErrorLogger is an optional io.Writer which errors are written to. Note
	// that these are request-specific errors (e.g. error while writing to the
	// client). Almost all template execution errors will cause a panic.
	ErrorLogger io.Writer
}

// Loader is a template loader and executor. This should be created as a
// global variable to execution speed.
type Loader struct {
	conf *Config
	fs   *pongo2.TemplateSet
	ts   time.Time
}

// Render is used to render a specific template, where "path" is the path
// within the provided Config.Loader. "ctx" is the extra context which can
// be provided and loaded within your template, however it is not required.
// All ctx keys will be loaded at the base level (in the template, you can
// just use "{{ yourvar }}"). In addition to this, there are a few predefined
// ctx keys:
//
//	url     -> request.URL
//	cachets -> The timestamp of when the loader was defined. This is useful
//	           to append at the end of your css/js/etc as a way of allowing
//	           the browser to not use the same cache after the application
//	           has been recompiled/restarted.
//
// ctx keys can be overridden. The priority is:
//  1. Context defined via Render().
//  2. Context defined via the default context function.
//  3. Default defined context by the package, mentioned above.
func (ld *Loader) Render(w http.ResponseWriter, r *http.Request, path string, rctx map[string]interface{}) {
	var atmpl *pongo2.Template
	var err error

	if ld.conf.CacheParsed {
		atmpl, err = ld.fs.FromCache(path)
	} else {
		atmpl, err = ld.fs.FromFile(path)
	}

	var orig *pongo2.Error

	if errors.As(err, &orig) {
		if os.IsNotExist(orig.OrigError) {
			if ld.conf.NotFoundHandler != nil {
				ld.conf.NotFoundHandler(w, r)
				return
			}

			panic(err)
		}
	}

	tpl := pongo2.Must(atmpl, err)

	var ctx map[string]interface{}

	if ld.conf.DefaultCtx != nil {
		ctx = ld.conf.DefaultCtx(w, r)
	}

	switch {
	case ctx == nil && rctx != nil:
		ctx = rctx
	case ctx == nil:
		ctx = make(map[string]interface{})
	case rctx != nil:
		for key := range rctx {
			ctx[key] = rctx[key]
		}
	}

	if _, ok := ctx["url"]; !ok {
		ctx["url"] = r.URL
	}
	if _, ok := ctx["cachets"]; !ok {
		ctx["cachets"] = ld.ts.Unix()
	}

	w.Header().Set("Content-Type", "text/html")

	err = tpl.ExecuteWriter(ctx, w)
	if err != nil {
		var pongoErr *pongo2.Error

		if errors.As(err, &pongoErr) {
			panic(err)
		}

		fmt.Fprint(ld.conf.ErrorLogger, "error: "+err.Error())
	}
}

// Router is a general interface which many common http routers should fit.
// See FileServer() for details.
type Router interface {
	Get(pattern string, h http.HandlerFunc)
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem. "router" matches any servemux style
// router which has a Get() method (e.g. go-chi/chi.Router).
//
// For example, mixing go-chi/chi + go-ricebox:
//
//	FileServer(r, "/static", rice.MustFindBox("static").HTTPBox())
func FileServer(router Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("url params not allowed in file server")
	}

	srv := http.StripPrefix(path, http.FileServer(root))

	if path != "/" && path[len(path)-1] != '/' {
		router.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	router.Get(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.ServeHTTP(w, r)
	}))
}

// JSONEscapeHTMLKey is a context key which can be used with JSON() to set
// HTML escaping to true.
const JSONEscapeHTMLKey = "JSONEscapeHTML"

// JSON marshals 'v' to JSON, and setting the Content-Type as application/json.
// Note that this does NOT auto-escape HTML. If you would like to escape HTML,
// set the JSONEscapeHTMLKey context value to true.
//
// JSON also supports prettification when the origin request has "?pretty=true"
// or similar.
func JSON(w http.ResponseWriter, r *http.Request, v interface{}) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)

	if escape, ok := r.Context().Value(JSONEscapeHTMLKey).(bool); ok && escape {
		enc.SetEscapeHTML(escape)
	}

	if pretty, _ := strconv.ParseBool(r.FormValue("pretty")); pretty {
		enc.SetIndent("", "    ")
	}

	if err := enc.Encode(v); err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(buf.Bytes())
}

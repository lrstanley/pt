// Copyright (c) Liam Stanley <me@liamstanley.io>. All rights reserved. Use
// of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package pt

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/flosch/pongo2/v5"
)

func init() {
	pongo2.RegisterFilter("json", filterJSON)
}

func filterJSON(in, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)

	// This doesn't need to be done, as pongo2 by default will escape vars.
	// enc.SetEscapeHTML(true)

	if args := strings.ToLower(param.String()); args == "pretty" {
		enc.SetIndent("", "    ")
	} else if args != "" {
		enc.SetIndent("", args)
	}

	if err := enc.Encode(in.Interface()); err != nil {
		return nil, &pongo2.Error{Sender: "filter:json", OrigError: err}
	}

	return pongo2.AsValue(b.String()), nil
}

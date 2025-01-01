// Copyright (c) Liam Stanley <liam@liam.sh>. All rights reserved. Use of
// this source code is governed by the MIT license that can be found in
// the LICENSE file.

package pt

import (
	"log"
	"net/http"
)

// Error logs the given error, as well as optionally returns the error back to
// the connection.
func Error(logger *log.Logger, w http.ResponseWriter, code int, err error, show bool) {
	if logger != nil {
		logger.Printf("http error: "+err.Error(), err)
	}

	if show {
		http.Error(w, "error: "+err.Error(), code)
		return
	}
	http.Error(w, http.StatusText(code), code)
}

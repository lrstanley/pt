// Copyright (c) Liam Stanley <me@liamstanley.io>. All rights reserved. Use
// of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package pt

import (
	"bytes"
	"io"
	"path/filepath"
)

type memLoader struct {
	loaderFunc func(path string) ([]byte, error)
}

func (m memLoader) Abs(base, name string) string {
	if filepath.IsAbs(name) || base == "" {
		return name
	}

	if name == "" {
		return base
	}

	return filepath.Dir(base) + string(filepath.Separator) + name
}

func (m memLoader) Get(path string) (io.Reader, error) {
	data, err := m.loaderFunc(path)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(data), nil
}

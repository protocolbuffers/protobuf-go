// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textfuzz

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func Test(t *testing.T) {
	dir, err := os.Open("corpus")
	if err != nil {
		t.Fatal(err)
	}
	infos, err := dir.Readdir(0)
	if err != nil {
		t.Fatal(err)

	}
	for _, info := range infos {
		name := info.Name()
		t.Run(name, func(t *testing.T) {
			b, err := ioutil.ReadFile(filepath.Join("corpus", name))
			if err != nil {
				t.Fatal(err)
			}
			Fuzz(b)
		})
	}
}

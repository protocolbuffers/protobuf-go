// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package depv1 exists to depend on github.com/golang/protobuf.
//
// We include this dependency to ensure that any program using
// APIv2 also uses a sufficiently new version of APIv1. At some
// point in the future when old versions of APIv1 are no longer
// of concern, we may drop this dependency.
package depv1

// TODO: Delete this dependency when it no longer serves a purpose.
import _ "github.com/golang/protobuf/proto"

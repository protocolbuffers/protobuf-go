// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package internal_gengo

import "google.golang.org/protobuf/compiler/protogen"

// Support for go "encoding/json" to use protojson specific Marshal/Unmarshaling
func genJSONFunctions(g *protogen.GeneratedFile, f *fileInfo, m *messageInfo) {
	g.P("func (p *", m.GoIdent.GoName, ") MarshalJSON() ([]byte, error) {")
	g.P("	return ", protojsonPackage.Ident("Marshal"), "(p)")
	g.P("}")
	g.P("func (p *", m.GoIdent.GoName, ") UnmarshalJSON(data []byte) error {")
	g.P("	return ", protojsonPackage.Ident("Unmarshal"), "(data, p)")
	g.P("}")
}

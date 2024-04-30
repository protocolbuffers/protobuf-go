// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package protopath provides functionality for
// representing a sequence of protobuf reflection operations on a message.
package protopath

import (
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

type parseState int

const (
	// needRoot expects either an identifier for an implicit root or '(' to start
	// parsing the message descriptor string.
	needRoot parseState = iota
	// needRootDescriptor expects the beginning of a qualified name for the message descriptor's FullName.
	needRootDescriptor
	// neetRootClose has seen at least one identifier and is not just past a '.'.
	needRootClose
	// expects one of '[', '.', or eof
	needAccessor
	// expects either '.' or eof
	needFieldAccessor
	// expects an identifier after a '.'
	needFieldName
	// expects a value and ']' for a map, or explicitly a natural and ']' for a list.
	needIndex
	// expects a ']' to complete an index operation
	needIndexClose
)

// returns whether the parser should accept what it's gotten so far as a valid path.
func (s parseState) isTerminal() bool {
	return s == needRoot || s == needAccessor || s == needFieldAccessor
}

// When the parser encounters an unexpected token, it needs a description of
// what is expected of its current state.
func (s parseState) expect() string {
	switch s {
	case needRoot:
		return "first field or '(' for full name"
	case needRootDescriptor:
		return "next name fragment of message descriptor's full name"
	case needRootClose:
		return "either '.' for next full name fragment or ')'"
	case needAccessor:
		return "one of '[', '.', or eof"
	case needFieldAccessor:
		return "either '.' or eof following root"
	case needFieldName:
		return "field name following '.'"
	case needIndex:
		return "index value for map or list"
	case needIndexClose:
		return "']'"
	default:
		return "unknown state"
	}
}

type parser struct {
	s     *scanner
	state parseState
	path  Path
	// The root message descriptor
	md protoreflect.MessageDescriptor
	// The descriptor for the field the parser's state is at right now.
	// This can be a regular message/field, a repeated field, or a map.
	desc protoreflect.Descriptor
	// when parsing a qualified name, these are the identifiers between '.'s.
	qnameFragments protoreflect.FullName
}

func newParser(md protoreflect.MessageDescriptor, path string) *parser {
	return &parser{
		s:    &scanner{buf: []byte(path)},
		path: Path{Root(md)},
		md:   md,
		desc: md,
	}
}

func (p *parser) unexpected(pos int, got string) error {
	return fmt.Errorf("%sgot %s", p.showState(pos), got)
}

func (p *parser) oparen(t *token) error {
	if p.state != needRoot {
		return p.unexpected(t.Pos, "'('")
	}
	p.state = needRootDescriptor
	return nil
}

func (p *parser) cparen(t *token) error {
	if p.state != needRootClose {
		return p.unexpected(t.Pos, "')'")
	}
	wantName := p.md.FullName()
	gotName := p.qnameFragments
	if wantName != gotName {
		return fmt.Errorf("%sat ')' got root message descriptor %q, want %q", p.showState(t.Pos), gotName, wantName)
	}
	p.state = needFieldAccessor
	// The path already has its root as part of parser initialization.
	return nil
}

func (p *parser) showState(pos int) string {
	prefix := fmt.Sprintf("column %d: ", pos)
	whitespace := strings.Repeat(" ", len(prefix)+pos)
	spanLength := p.s.pos - pos
	endcap := "|"
	// If the range to point out is only a single character, don't add a second character to denote
	// the end of the range.
	if spanLength < 2 {
		endcap = ""
	}
	dashes := spanLength - 2
	if dashes < 0 {
		dashes = 0
	}
	filler := strings.Repeat("-", dashes)
	attention := fmt.Sprintf("%s^%s%s", whitespace, filler, endcap)
	return fmt.Sprintf("{expect %s}\n%s%s\n%s\n", p.state.expect(), prefix, string(p.s.buf), attention)
}

func (p *parser) accessIdent(pos int, id string) error {
	m := p.desc
	// This can be a Message field in a message, so extract that message.
	if fd, ok := m.(protoreflect.FieldDescriptor); ok {
		// If the cursor is at a map and the identifier uses an internal field "key" or "value", then
		// disallow that.
		if fd.IsMap() && (id == "key" || id == "value") {
			return fmt.Errorf("%smap internal field %q may not be traversed", p.showState(pos), id)
		}
		m = fd.Message()
	}
	md, ok := m.(protoreflect.MessageDescriptor)
	if !ok {
		return fmt.Errorf("%sexpected message descriptor to access with field %q, got %T", p.showState(pos), id, p.desc)
	}
	fd := md.Fields().ByTextName(id)
	if fd == nil {
		return fmt.Errorf("%sfield %q not in message descriptor that has fields %v", p.showState(pos), id, md.Fields())
	}
	p.desc = fd
	p.state = needAccessor
	p.path = append(p.path, FieldAccess(fd))
	return nil
}

type forAccess interface {
	castKey(keyKind protoreflect.Kind) (protoreflect.MapKey, bool)
	castInt() (int, bool)
}

type stringForMapKey struct {
	s string
}

type numberForAccess struct {
	lit string
}

type boolAccess struct {
	b bool
}

func (b *boolAccess) castKey(keyKind protoreflect.Kind) (protoreflect.MapKey, bool) {
	if keyKind == protoreflect.BoolKind {
		return protoreflect.ValueOfBool(b.b).MapKey(), true
	}
	return protoreflect.MapKey{}, false
}

func (b *boolAccess) castInt() (int, bool) {
	return 0, false
}

func (b *boolAccess) String() string {
	return strconv.FormatBool(b.b)
}

func (s *stringForMapKey) castKey(keyKind protoreflect.Kind) (protoreflect.MapKey, bool) {
	if keyKind == protoreflect.StringKind {
		return protoreflect.ValueOfString(s.s).MapKey(), true
	}
	return protoreflect.MapKey{}, false
}

func (s *stringForMapKey) castInt() (int, bool) {
	return 0, false
}

func (s *stringForMapKey) String() string {
	return strconv.Quote(s.s)
}

func (n *numberForAccess) castInt() (int, bool) {
	i, err := strconv.ParseInt(n.lit, 0, 64)
	if err != nil {
		return 0, false
	}
	return int(i), true
}

func (n *numberForAccess) castKey(kind protoreflect.Kind) (protoreflect.MapKey, bool) {
	switch kind {
	case protoreflect.Int32Kind:
		i32, err := strconv.ParseInt(n.lit, 0, 32)
		if err != nil {
			return protoreflect.MapKey{}, false
		}
		return protoreflect.ValueOfInt32(int32(i32)).MapKey(), true
	case protoreflect.Int64Kind:
		i64, err := strconv.ParseInt(n.lit, 0, 64)
		if err != nil {
			return protoreflect.MapKey{}, false
		}
		return protoreflect.ValueOfInt64(i64).MapKey(), true
	case protoreflect.Uint32Kind:
		u32, err := strconv.ParseUint(n.lit, 0, 32)
		if err != nil {
			return protoreflect.MapKey{}, false
		}
		return protoreflect.ValueOfUint32(uint32(u32)).MapKey(), true
	case protoreflect.Uint64Kind:
		u64, err := strconv.ParseUint(n.lit, 0, 64)
		if err != nil {
			return protoreflect.MapKey{}, false
		}
		return protoreflect.ValueOfUint64(u64).MapKey(), true
	default:
		return protoreflect.MapKey{}, false
	}
}

func (n *numberForAccess) String() string {
	return n.lit
}

// Index a map or list with the given literal.
func (p *parser) accessValue(pos int, lit forAccess) error {
	fd, ok := p.desc.(protoreflect.FieldDescriptor)
	if !ok {
		return fmt.Errorf("%sexpected field descriptor to access with value %v, got %T", p.showState(pos), lit, p.desc)
	}
	if fd.Cardinality() != protoreflect.Repeated {
		return fmt.Errorf("position %d: expected field descriptor with repeated cardinality "+
			"(map or list) to access with value %v, got %T", pos, lit, p.desc)
	}
	if fd.IsMap() {
		mk, ok := lit.castKey(fd.MapKey().Kind())
		if !ok {
			return fmt.Errorf("%scannot index map with key kind %v with key %v", p.showState(pos),
				fd.MapKey().Kind(), lit)
		}
		p.desc = fd.MapValue().Message()
		p.path = append(p.path, MapIndex(mk))
	} else {
		i, ok := lit.castInt()
		if !ok {
			// although bool is integral, you can't index a list by bool.
			return fmt.Errorf("%scannot index list %T with non-integral type %v", p.showState(pos), fd, lit)
		}
		if i < 0 {
			return fmt.Errorf("%scannot index list %T with negative index %d", p.showState(pos), fd, i)
		}
		p.desc = fd.Message()
		p.path = append(p.path, ListIndex(i))
	}
	p.state = needIndexClose
	return nil
}

func (p *parser) ident(t *token) error {
	if p.state == needRootDescriptor {
		p.state = needRootClose
		p.qnameFragments = p.qnameFragments.Append(protoreflect.Name(t.Text))
		return nil
	}
	if p.state == needRoot || p.state == needFieldName {
		// implicit  '(' root ')' '.' when needRoot.
		return p.accessIdent(t.Pos, t.Text)
	}
	if p.state == needIndex {
		if t.Text == "true" {
			return p.accessValue(t.Pos, &boolAccess{b: true})
		}
		if t.Text == "false" {
			return p.accessValue(t.Pos, &boolAccess{b: false})
		}
		return fmt.Errorf("%sexpected value for index, not an identifier %q", p.showState(t.Pos), t.Text)
	}

	return p.unexpected(t.Pos, fmt.Sprintf("identifier %q", t.Text))
}

func (p *parser) intlit(t *token) error {
	if p.state != needIndex {
		return p.unexpected(t.Pos, fmt.Sprintf("integer %s", t.Text))
	}
	return p.accessValue(t.Pos, &numberForAccess{lit: t.Text})
}

func (p *parser) strlit(t *token) error {
	if p.state != needIndex {
		return p.unexpected(t.Pos, fmt.Sprintf("string %q", t.Text))
	}
	return p.accessValue(t.Pos, &stringForMapKey{s: t.Text})
}

func (p *parser) obrack(t *token) error {
	if p.state != needAccessor {
		return p.unexpected(t.Pos, "'['")
	}
	p.state = needIndex
	return nil
}

func (p *parser) cbrack(t *token) error {
	// the index has already extended the path, so switch state.
	if p.state != needIndexClose {
		return p.unexpected(t.Pos, "']'")
	}
	p.state = needAccessor
	return nil
}

func (p *parser) dot(t *token) error {
	switch p.state {
	case needRootClose:
		p.state = needRootDescriptor
	case needAccessor:
		p.state = needFieldName
	case needFieldAccessor:
		p.state = needFieldName
	default:
		return p.unexpected(t.Pos, "'.'")
	}
	return nil
}

func (p *parser) step(tok *token) error {
	switch tok.Kind {
	case oparen: // ought to be the first token
		return p.oparen(tok)
	case cparen: // ought to follow a qualified name.
		return p.cparen(tok)
	case obrack: // ought to follow a field and not a root.
		return p.obrack(tok)
	case cbrack: // ought to follow an index/mapkey.
		return p.cbrack(tok)
	case ident: // ought to be a qualified name piece or field
		return p.ident(tok)
	case intlit: // ought to be a mapkey or list index.
		return p.intlit(tok)
	case strlit: // ought only be a mapkey.
		return p.strlit(tok)
	case dot: // ought to be a qualified name or field access.
		return p.dot(tok)
	case illegal:
		return fmt.Errorf("found illegal token %s at position %d", tok.Text, tok.Pos)
	default:
		return fmt.Errorf("unknown token kind %d", tok.Kind)
	}
}

// ParsePath translates a human-readable representation of a path into a Path.
//
// An empty path is an empty string.
// A field access step is path '.' identifier
// A map index step is path '[' natural ']'
// A root step is '(' msg.Descriptor().String() ')'
//
// If the path does not start with '(' then the root step is implicitly for the given message.
// The parser is "type aware" to distinguish lists and maps keyed by numbers.
func ParsePath(md protoreflect.MessageDescriptor, path string) (Path, error) {
	if md == nil {
		return nil, fmt.Errorf("ParsePath: message descriptor must be non-nil")
	}
	p := newParser(md, path)
	for {
		tok := p.s.scan()
		if tok.Kind == eof {
			if p.state.isTerminal() {
				return p.path, nil
			}
			return nil, fmt.Errorf("finished parsing in state that expects %s", p.state.expect())
		}

		if err := p.step(tok); err != nil {
			return nil, fmt.Errorf("path %q parse failure: %v", path, err)
		}
	}
}

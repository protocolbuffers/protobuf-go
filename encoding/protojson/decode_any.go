// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protojson

import (
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/internal/encoding/json"
	"google.golang.org/protobuf/internal/errors"
	"google.golang.org/protobuf/internal/genid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// anyWire is the binary encoding of a message as a linked list of chunks.
// Nested Any values are wrapped by prepending a small envelope chunk. The
// payload chunks themselves are not copied until the outermost Any is stored,
// so a chain of N Any messages is encoded in O(N) time and allocations.
type anyWire struct {
	chunk []byte
	next  *anyWire
	n     int
}

func flatWire(b []byte) anyWire {
	if len(b) == 0 {
		return anyWire{}
	}
	return anyWire{chunk: b, n: len(b)}
}

func consWire(head []byte, rest anyWire) anyWire {
	if rest.n == 0 {
		return anyWire{chunk: head, n: len(head)}
	}
	restCopy := rest
	return anyWire{chunk: head, next: &restCopy, n: len(head) + rest.n}
}

// bytes returns the concatenated encoding. A single chunk is returned
// directly; a chain is copied once into a fresh buffer.
func (w anyWire) bytes() []byte {
	if w.n == 0 {
		return nil
	}
	if w.next == nil {
		return w.chunk
	}
	b := make([]byte, 0, w.n)
	for p := &w; p != nil; p = p.next {
		b = append(b, p.chunk...)
	}
	return b
}

// wrapAnyWire returns the wire encoding of an Any message with the given
// type URL and embedded message bytes. Field 2 is omitted when inner is
// empty, matching proto.Marshal of an Any that has only type_url set.
func wrapAnyWire(typeURL string, inner anyWire) anyWire {
	var head []byte
	head = protowire.AppendTag(head, genid.Any_TypeUrl_field_number, protowire.BytesType)
	head = protowire.AppendString(head, typeURL)
	if inner.n > 0 {
		head = protowire.AppendTag(head, genid.Any_Value_field_number, protowire.BytesType)
		head = protowire.AppendVarint(head, uint64(inner.n))
	}
	return consWire(head, inner)
}

func (d decoder) burnRecursion() (decoder, error) {
	d.opts.RecursionLimit--
	if d.opts.RecursionLimit < 0 {
		return d, errors.New("exceeded max recursion depth")
	}
	return d, nil
}

func (d decoder) sliceDecoder(start, end int) decoder {
	return decoder{d.Range(start, end), d.opts, d.cache}
}

// jsonField is one object member captured while resolving an Any whose
// "@type" is not the first field. The value is parsed once; it is not
// tokenized again when the surrounding Any is itself nested.
type jsonField struct {
	name    string
	nameTok json.Token
	val     jsonNode
}

// jsonNode is a JSON value parsed by a single walk of the decoder.
type jsonNode struct {
	kind     json.Kind
	tok      json.Token // opening token, or the scalar token
	closeTok json.Token // ObjectClose, for objects
	fields   []jsonField
	start    int
	end      int
}

func (d decoder) readJSONNode(depth int) (jsonNode, error) {
	tok, err := d.Peek()
	if err != nil {
		return jsonNode{}, err
	}
	switch tok.Kind() {
	case json.ObjectOpen, json.ArrayOpen:
		// Bound the parse itself. Any-to-Any recursion is also counted when
		// the value is later unmarshaled; this check only keeps a deeply
		// nested "@type"-last document from overflowing the stack before that.
		if depth > d.opts.RecursionLimit {
			return jsonNode{}, errors.New("exceeded max recursion depth")
		}
		if tok.Kind() == json.ObjectOpen {
			return d.readObjectNode(depth)
		}
		return d.readArrayNode(depth)
	case json.Null, json.Bool, json.Number, json.String:
		start := tok.Pos()
		tok, err = d.Read()
		if err != nil {
			return jsonNode{}, err
		}
		return jsonNode{kind: tok.Kind(), tok: tok, start: start, end: d.Offset()}, nil
	default:
		return jsonNode{}, d.unexpectedTokenError(tok)
	}
}

func (d decoder) readObjectNode(depth int) (jsonNode, error) {
	open, err := d.Read()
	if err != nil {
		return jsonNode{}, err
	}
	var fields []jsonField
	for {
		tok, err := d.Read()
		if err != nil {
			return jsonNode{}, err
		}
		switch tok.Kind() {
		case json.ObjectClose:
			return jsonNode{
				kind:     json.ObjectOpen,
				tok:      open,
				closeTok: tok,
				fields:   fields,
				start:    open.Pos(),
				end:      d.Offset(),
			}, nil
		case json.Name:
			val, err := d.readJSONNode(depth + 1)
			if err != nil {
				return jsonNode{}, err
			}
			fields = append(fields, jsonField{name: tok.Name(), nameTok: tok, val: val})
		default:
			return jsonNode{}, d.unexpectedTokenError(tok)
		}
	}
}

func (d decoder) readArrayNode(depth int) (jsonNode, error) {
	open, err := d.Read()
	if err != nil {
		return jsonNode{}, err
	}
	// Arrays are retained only as a byte span. Well-known Any values are
	// objects (or scalars); an array is reparsed at most once, for a
	// ListValue or a repeated field, after the enclosing type is known.
	for {
		tok, err := d.Peek()
		if err != nil {
			return jsonNode{}, err
		}
		if tok.Kind() == json.ArrayClose {
			if _, err := d.Read(); err != nil {
				return jsonNode{}, err
			}
			return jsonNode{kind: json.ArrayOpen, tok: open, start: open.Pos(), end: d.Offset()}, nil
		}
		if _, err := d.readJSONNode(depth + 1); err != nil {
			return jsonNode{}, err
		}
	}
}

func (d decoder) unmarshalAny(m protoreflect.Message) error {
	_, err := d.decodeAny(m, true)
	return err
}

// decodeAny unmarshals a google.protobuf.Any JSON object.
// When materialize is false, m is left unset and the returned wire is the
// encoding of the Any. Callers that embed this Any in another Any use that
// wire instead of copying m's value field.
func (d decoder) decodeAny(m protoreflect.Message, materialize bool) (anyWire, error) {
	start, err := d.Peek()
	if err != nil {
		return anyWire{}, err
	}
	if start.Kind() != json.ObjectOpen {
		return anyWire{}, d.unexpectedTokenError(start)
	}
	if _, err := d.Read(); err != nil {
		return anyWire{}, err
	}

	tok, err := d.Read()
	if err != nil {
		return anyWire{}, err
	}
	switch tok.Kind() {
	case json.ObjectClose:
		// An empty JSON object is an empty Any.
		return anyWire{}, nil
	case json.Name:
	default:
		return anyWire{}, d.unexpectedTokenError(tok)
	}
	if tok.Name() == "@type" {
		return d.decodeAnyTypeFirst(m, start, materialize)
	}
	return d.decodeAnyTypeLater(m, start, tok, materialize)
}

func (d decoder) readTypeURL() (string, json.Token, error) {
	tok, err := d.Read()
	if err != nil {
		return "", json.Token{}, err
	}
	if tok.Kind() != json.String {
		return "", json.Token{}, d.newError(tok.Pos(), `@type field value is not a string: %v`, tok.RawString())
	}
	s := tok.ParsedString()
	if s == "" {
		return "", json.Token{}, d.newError(tok.Pos(), `@type field contains empty value`)
	}
	return s, tok, nil
}

func (d decoder) newAnyEmbed(typeURL string, typeTok json.Token) (protoreflect.Message, bool, error) {
	emt, err := d.opts.Resolver.FindMessageByURL(typeURL)
	if err != nil {
		return nil, false, d.newError(typeTok.Pos(), "unable to resolve %v: %q", typeTok.RawString(), err)
	}
	em := emt.New()
	isWKT := wellKnownTypeUnmarshaler(em.Descriptor().FullName()) != nil
	return em, isWKT, nil
}

func (d decoder) marshalEmbed(em protoreflect.Message, pos int) (anyWire, error) {
	b, err := proto.MarshalOptions{
		AllowPartial:  true, // No need to check required fields inside an Any.
		Deterministic: true,
	}.Marshal(em.Interface())
	if err != nil {
		return anyWire{}, d.newError(pos, "error in marshaling Any.value field: %v", err)
	}
	return flatWire(b), nil
}

func (d decoder) finishAny(m protoreflect.Message, typeURL string, embed anyWire, materialize bool) (anyWire, error) {
	full := wrapAnyWire(typeURL, embed)
	if !materialize {
		return full, nil
	}
	fds := m.Descriptor().Fields()
	m.Set(fds.ByNumber(genid.Any_TypeUrl_field_number), protoreflect.ValueOfString(typeURL))
	m.Set(fds.ByNumber(genid.Any_Value_field_number), protoreflect.ValueOfBytes(embed.bytes()))
	return full, nil
}

// decodeAnyTypeFirst handles the common shape where "@type" is the first
// field. The rest of the object is unmarshaled directly from the decoder.
func (d decoder) decodeAnyTypeFirst(m protoreflect.Message, start json.Token, materialize bool) (anyWire, error) {
	typeURL, typeTok, err := d.readTypeURL()
	if err != nil {
		return anyWire{}, err
	}
	em, isWKT, err := d.newAnyEmbed(typeURL, typeTok)
	if err != nil {
		return anyWire{}, err
	}
	var embed anyWire
	if isWKT {
		embed, err = d.readWKTStreaming(em)
		if err != nil {
			return anyWire{}, err
		}
	} else {
		// The embedded message is not a well-known type, so the current
		// object is that message plus "@type", which was already consumed.
		// unmarshalOpenObject does not decrement; count this message once.
		d, err = d.burnRecursion()
		if err != nil {
			return anyWire{}, err
		}
		if err := d.unmarshalOpenObject(em, true, true); err != nil {
			return anyWire{}, err
		}
		embed, err = d.marshalEmbed(em, start.Pos())
		if err != nil {
			return anyWire{}, err
		}
	}
	return d.finishAny(m, typeURL, embed, materialize)
}

// readWKTStreaming reads the remaining fields of a well-known Any object
// whose "@type" was already consumed. RecursionLimit is decremented once
// before the "value" is unmarshaled. It is not decremented for a non-WKT
// embed, which goes through unmarshalMessage / unmarshalOpenObject.
func (d decoder) readWKTStreaming(em protoreflect.Message) (anyWire, error) {
	unmarshal := wellKnownTypeUnmarshaler(em.Descriptor().FullName())
	isAny := em.Descriptor().FullName() == genid.Any_message_fullname
	isEmpty := em.Descriptor().FullName() == genid.Empty_message_fullname
	var (
		found bool
		embed anyWire
	)
	for {
		tok, err := d.Read()
		if err != nil {
			return anyWire{}, err
		}
		switch tok.Kind() {
		case json.ObjectClose:
			if !found && !isEmpty {
				return anyWire{}, d.newError(tok.Pos(), `missing "value" field`)
			}
			if isAny {
				return embed, nil
			}
			return d.marshalEmbed(em, tok.Pos())
		case json.Name:
			switch tok.Name() {
			case "@type":
				return anyWire{}, d.newError(tok.Pos(), `duplicate "@type" field`)
			case "value":
				if found {
					return anyWire{}, d.newError(tok.Pos(), `duplicate "value" field`)
				}
				found = true
				d, err = d.burnRecursion()
				if err != nil {
					return anyWire{}, err
				}
				if isAny {
					embed, err = d.decodeAny(em, false)
					if err != nil {
						return anyWire{}, err
					}
					continue
				}
				if err := unmarshal(d, em); err != nil {
					return anyWire{}, err
				}
			default:
				if d.opts.DiscardUnknown {
					if err := d.skipJSONValue(); err != nil {
						return anyWire{}, err
					}
					continue
				}
				return anyWire{}, d.newError(tok.Pos(), "unknown field %v", tok.RawString())
			}
		default:
			return anyWire{}, d.unexpectedTokenError(tok)
		}
	}
}

// decodeAnyTypeLater handles an Any object whose first field is not "@type".
// Every sibling value is parsed once into a tree. If the embedded message is
// itself an Any, that tree is interpreted in place. Re-tokenizing each
// ancestor's "value" would make "@type"-last nesting quadratic.
func (d decoder) decodeAnyTypeLater(m protoreflect.Message, start, firstName json.Token, materialize bool) (anyWire, error) {
	var (
		typeURL string
		typeTok json.Token
		fields  []jsonField
	)
	nameTok := firstName
	for {
		if nameTok.Name() == "@type" {
			if typeURL != "" {
				return anyWire{}, d.newError(nameTok.Pos(), `duplicate "@type" field`)
			}
			var err error
			typeURL, typeTok, err = d.readTypeURL()
			if err != nil {
				return anyWire{}, err
			}
		} else {
			val, err := d.readJSONNode(1)
			if err != nil {
				return anyWire{}, err
			}
			fields = append(fields, jsonField{name: nameTok.Name(), nameTok: nameTok, val: val})
		}
		tok, err := d.Read()
		if err != nil {
			return anyWire{}, err
		}
		switch tok.Kind() {
		case json.ObjectClose:
			if typeURL == "" {
				if d.opts.DiscardUnknown {
					return anyWire{}, nil
				}
				return anyWire{}, d.newError(start.Pos(), `missing "@type" field`)
			}
			return d.applyAnyFields(m, start.Pos(), d.Offset(), tok.Pos(), typeURL, typeTok, fields, materialize)
		case json.Name:
			nameTok = tok
		default:
			return anyWire{}, d.unexpectedTokenError(tok)
		}
	}
}

func (d decoder) applyAnyFields(m protoreflect.Message, objStart, objEnd, closePos int, typeURL string, typeTok json.Token, fields []jsonField, materialize bool) (anyWire, error) {
	em, isWKT, err := d.newAnyEmbed(typeURL, typeTok)
	if err != nil {
		return anyWire{}, err
	}
	if !isWKT {
		// One reparse of this object. Nested Any values inside it are decoded
		// by the single-pass path, so the cost stays linear in the input size.
		sub := d.sliceDecoder(objStart, objEnd)
		if err := sub.unmarshalMessage(em, true); err != nil {
			return anyWire{}, err
		}
		embed, err := d.marshalEmbed(em, objStart)
		if err != nil {
			return anyWire{}, err
		}
		return d.finishAny(m, typeURL, embed, materialize)
	}

	unmarshal := wellKnownTypeUnmarshaler(em.Descriptor().FullName())
	isAny := em.Descriptor().FullName() == genid.Any_message_fullname
	isEmpty := em.Descriptor().FullName() == genid.Empty_message_fullname
	var (
		found bool
		embed anyWire
	)
	for _, f := range fields {
		switch f.name {
		case "value":
			if found {
				return anyWire{}, d.newError(f.nameTok.Pos(), `duplicate "value" field`)
			}
			found = true
			d, err = d.burnRecursion()
			if err != nil {
				return anyWire{}, err
			}
			if isAny {
				embed, err = d.decodeAnyFromNode(em, f.val, false)
				if err != nil {
					return anyWire{}, err
				}
				continue
			}
			sub := d.sliceDecoder(f.val.start, f.val.end)
			if err := unmarshal(sub, em); err != nil {
				return anyWire{}, err
			}
		default:
			if d.opts.DiscardUnknown {
				continue
			}
			return anyWire{}, d.newError(f.nameTok.Pos(), "unknown field %v", f.nameTok.RawString())
		}
	}
	if !found && !isEmpty {
		return anyWire{}, d.newError(closePos, `missing "value" field`)
	}
	if isAny {
		return d.finishAny(m, typeURL, embed, materialize)
	}
	embed, err = d.marshalEmbed(em, objStart)
	if err != nil {
		return anyWire{}, err
	}
	return d.finishAny(m, typeURL, embed, materialize)
}

// decodeAnyFromNode unmarshals an Any JSON object that was already parsed.
func (d decoder) decodeAnyFromNode(m protoreflect.Message, node jsonNode, materialize bool) (anyWire, error) {
	if node.kind != json.ObjectOpen {
		return anyWire{}, d.unexpectedTokenError(node.tok)
	}
	if len(node.fields) == 0 {
		return anyWire{}, nil
	}
	var (
		typeURL   string
		typeTok   json.Token
		typeCount int
		fields    []jsonField
	)
	for _, f := range node.fields {
		if f.name != "@type" {
			fields = append(fields, f)
			continue
		}
		typeCount++
		if typeCount > 1 {
			return anyWire{}, d.newError(f.nameTok.Pos(), `duplicate "@type" field`)
		}
		if f.val.kind != json.String {
			return anyWire{}, d.newError(f.val.tok.Pos(), `@type field value is not a string: %v`, f.val.tok.RawString())
		}
		s := f.val.tok.ParsedString()
		if s == "" {
			return anyWire{}, d.newError(f.val.tok.Pos(), `@type field contains empty value`)
		}
		typeURL = s
		typeTok = f.val.tok
	}
	if typeURL == "" {
		if d.opts.DiscardUnknown {
			return anyWire{}, nil
		}
		return anyWire{}, d.newError(node.tok.Pos(), `missing "@type" field`)
	}
	return d.applyAnyFields(m, node.start, node.end, node.closeTok.Pos(), typeURL, typeTok, fields, materialize)
}

// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package text

import (
	"bytes"
	"io"
	"math"
	"strconv"

	"google.golang.org/protobuf/internal/errors"
)

// marshalNumber encodes v as either a Bool, Int, Uint, Float32, or Float64.
func (p *encoder) marshalNumber(v Value) error {
	var err error
	p.out, err = appendNumber(p.out, v)
	return err
}
func appendNumber(out []byte, v Value) ([]byte, error) {
	if len(v.raw) > 0 {
		switch v.Type() {
		case Bool, Int, Uint, Float32, Float64:
			return append(out, v.raw...), nil
		}
	}
	switch v.Type() {
	case Bool:
		if b, _ := v.Bool(); b {
			return append(out, "true"...), nil
		} else {
			return append(out, "false"...), nil
		}
	case Int:
		return strconv.AppendInt(out, int64(v.num), 10), nil
	case Uint:
		return strconv.AppendUint(out, uint64(v.num), 10), nil
	case Float32:
		return appendFloat(out, v, 32)
	case Float64:
		return appendFloat(out, v, 64)
	default:
		return nil, errors.New("invalid type %v, expected bool or number", v.Type())
	}
}

func appendFloat(out []byte, v Value, bitSize int) ([]byte, error) {
	switch n := math.Float64frombits(v.num); {
	case math.IsNaN(n):
		return append(out, "nan"...), nil
	case math.IsInf(n, +1):
		return append(out, "inf"...), nil
	case math.IsInf(n, -1):
		return append(out, "-inf"...), nil
	default:
		return strconv.AppendFloat(out, n, 'g', -1, bitSize), nil
	}
}

// These regular expressions were derived by reverse engineering the C++ code
// in tokenizer.cc and text_format.cc.
var (
	literals = map[string]interface{}{
		// These exact literals are the ones supported in C++.
		// In C++, a 1-bit unsigned integers is also allowed to represent
		// a boolean. This is handled in Value.Bool.
		"t":     true,
		"true":  true,
		"True":  true,
		"f":     false,
		"false": false,
		"False": false,

		// C++ permits "-nan" and the case-insensitive variants of these.
		// However, Go continues to be case-sensitive.
		"nan":  math.NaN(),
		"inf":  math.Inf(+1),
		"-inf": math.Inf(-1),
	}
)

// unmarshalNumber decodes a Bool, Int, Uint, or Float64 from the input.
func (p *decoder) unmarshalNumber() (Value, error) {
	v, n, err := consumeNumber(p.in)
	p.consume(n)
	return v, err
}

func consumeNumber(in []byte) (Value, int, error) {
	if len(in) == 0 {
		return Value{}, 0, io.ErrUnexpectedEOF
	}
	if v, n := matchLiteral(in); n > 0 {
		return rawValueOf(v, in[:n]), n, nil
	}

	num, ok := parseNumber(in)
	if !ok {
		return Value{}, 0, newSyntaxError("invalid %q as number or bool", errRegexp.Find(in))
	}

	if num.typ == numFloat {
		f, err := strconv.ParseFloat(string(num.value), 64)
		if err != nil {
			return Value{}, 0, err
		}
		return rawValueOf(f, in[:num.size]), num.size, nil
	}

	if num.neg {
		v, err := strconv.ParseInt(string(num.value), 0, 64)
		if err != nil {
			return Value{}, 0, err
		}
		return rawValueOf(v, num.value), num.size, nil
	}
	v, err := strconv.ParseUint(string(num.value), 0, 64)
	if err != nil {
		return Value{}, 0, err
	}
	return rawValueOf(v, num.value), num.size, nil
}

func matchLiteral(in []byte) (interface{}, int) {
	switch in[0] {
	case 't', 'T':
		rest := in[1:]
		if len(rest) == 0 || isDelim(rest[0]) {
			return true, 1
		}
		if n := matchStringWithDelim("rue", rest); n > 0 {
			return true, 4
		}
	case 'f', 'F':
		rest := in[1:]
		if len(rest) == 0 || isDelim(rest[0]) {
			return false, 1
		}
		if n := matchStringWithDelim("alse", rest); n > 0 {
			return false, 5
		}
	case 'n':
		if n := matchStringWithDelim("nan", in); n > 0 {
			return math.NaN(), 3
		}
	case 'i':
		if n := matchStringWithDelim("inf", in); n > 0 {
			return math.Inf(1), 3
		}
	case '-':
		if n := matchStringWithDelim("-inf", in); n > 0 {
			return math.Inf(-1), 4
		}
	}
	return nil, 0
}

func matchStringWithDelim(s string, b []byte) int {
	if !bytes.HasPrefix(b, []byte(s)) {
		return 0
	}

	n := len(s)
	if n < len(b) && !isDelim(b[n]) {
		return 0
	}
	return n
}

type numType uint8

const (
	numDec numType = (1 << iota) / 2
	numHex
	numOct
	numFloat
)

// number is the result of parsing out a valid number from parseNumber. It
// contains data for doing float or integer conversion via the strconv package.
type number struct {
	typ numType
	neg bool
	// Size of input taken up by the number. This may not be the same as
	// len(number.value).
	size int
	// Bytes for doing strconv.Parse{Float,Int,Uint} conversion.
	value []byte
}

// parseNumber constructs a number object from given input. It allows for the
// following patterns:
//   integer: ^-?([1-9][0-9]*|0[xX][0-9a-fA-F]+|0[0-7]*)
//   float: ^-?((0|[1-9][0-9]*)?([.][0-9]*)?([eE][+-]?[0-9]+)?[fF]?)
func parseNumber(input []byte) (number, bool) {
	var size int
	var neg bool
	typ := numDec

	s := input
	if len(s) == 0 {
		return number{}, false
	}

	// Optional -
	if s[0] == '-' {
		neg = true
		s = s[1:]
		size++
		if len(s) == 0 {
			return number{}, false
		}
	}

	// C++ allows for whitespace and comments in between the negative sign and
	// the rest of the number. This logic currently does not but is consistent
	// with v1.

	switch {
	case s[0] == '0':
		if len(s) > 1 {
			switch {
			case s[1] == 'x' || s[1] == 'X':
				// Parse as hex number.
				typ = numHex
				n := 2
				s = s[2:]
				for len(s) > 0 && (('0' <= s[0] && s[0] <= '9') ||
					('a' <= s[0] && s[0] <= 'f') ||
					('A' <= s[0] && s[0] <= 'F')) {
					s = s[1:]
					n++
				}
				if n == 2 {
					return number{}, false
				}
				size += n

			case '0' <= s[1] && s[1] <= '7':
				// Parse as octal number.
				typ = numOct
				n := 2
				s = s[2:]
				for len(s) > 0 && '0' <= s[0] && s[0] <= '7' {
					s = s[1:]
					n++
				}
				size += n
			}

			if typ&(numHex|numOct) > 0 {
				if len(s) > 0 && !isDelim(s[0]) {
					return number{}, false
				}
				return number{
					typ:   typ,
					size:  size,
					neg:   neg,
					value: input[:size],
				}, true
			}
		}
		s = s[1:]
		size++

	case '1' <= s[0] && s[0] <= '9':
		n := 1
		s = s[1:]
		for len(s) > 0 && '0' <= s[0] && s[0] <= '9' {
			s = s[1:]
			n++
		}
		size += n

	case s[0] == '.':
		// Handled below.

	default:
		return number{}, false
	}

	// . followed by 0 or more digits.
	if len(s) > 0 && s[0] == '.' {
		typ = numFloat
		n := 1
		s = s[1:]
		for len(s) > 0 && '0' <= s[0] && s[0] <= '9' {
			s = s[1:]
			n++
		}
		size += n
	}

	// e or E followed by an optional - or + and 1 or more digits.
	if len(s) >= 2 && (s[0] == 'e' || s[0] == 'E') {
		typ = numFloat
		s = s[1:]
		n := 1
		if s[0] == '+' || s[0] == '-' {
			s = s[1:]
			n++
			if len(s) == 0 {
				return number{}, false
			}
		}
		for len(s) > 0 && '0' <= s[0] && s[0] <= '9' {
			s = s[1:]
			n++
		}
		size += n
	}

	// At this point, input[:size] contains a valid number that can be converted
	// via strconv.Parse{Float,Int,Uint}.
	value := input[:size]

	// Optional suffix f or F for floats.
	if len(s) > 0 && (s[0] == 'f' || s[0] == 'F') {
		typ = numFloat
		s = s[1:]
		size++
	}

	// Check that next byte is a delimiter or it is at the end.
	if len(s) > 0 && !isDelim(s[0]) {
		return number{}, false
	}

	return number{
		typ:   typ,
		size:  size,
		neg:   neg,
		value: value,
	}, true
}

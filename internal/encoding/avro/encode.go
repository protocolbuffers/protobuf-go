package avro

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"math"

	"github.com/hamba/avro"
)

// Encoder provides methods to write out JSON constructs and values. The user is
// responsible for producing valid sequences of JSON constructs and values.
type Encoder struct {
	out []byte
	aw  *avro.Writer
	bw  *bufio.Writer
	buf *bytes.Buffer
}

// NewEncoder returns an Encoder.
func NewEncoder() *Encoder {
	// buf := bytes.NewBuffer(nil)
	// byteswriter := bufio.NewWriter(buf)
	return &Encoder{
		out: []byte{},
		// bw:  byteswriter,
		// aw:  avro.NewWriter(byteswriter, 0),
		// buf: buf,
	}
}

// Bytes returns the content of the written bytes.
func (e *Encoder) Bytes() []byte {
	return e.out
}

func (e *Encoder) writeByte(b byte) {
	e.out = append(e.out, b)
}

func (e *Encoder) encodeInt(i uint64) {
	if i == 0 {
		e.writeByte(0x00)
		return
	}

	for i > 0 {
		b := byte(i) & 0x7F
		i >>= 7

		if i != 0 {
			b |= 0x80
		}

		e.writeByte(b)
	}
}

// WriteNull writes out the null value.
func (e *Encoder) WriteNull() {
	e.writeByte(0x00)
}

/*
	Note all of the following function are assuming that all fields in the schema is nullable and the default value is null.
	So unless we are writing a null value, we should write a 1 to indicate the index followed by the value.
*/

// WriteBool writes out the given boolean value.
func (e *Encoder) WriteBool(b bool) {
	// Write the index of union type, assuming that the 0-index is the null value
	e.writeByte(0x02)
	if b {
		e.writeByte(0x01)
		return
	}
	e.writeByte(0x00)
}

// WriteInt writes an Int32 including the union type index
func (e *Encoder) WriteInt(i int64) {
	// Write the index of union type, assuming that the 0-index is the null value
	e.writeByte(0x02)
	v := uint64((uint32(i) << 1) ^ uint32(i>>31))
	e.encodeInt(v)
}

// WriteLong writes a Long including the union type index
func (e *Encoder) WriteLong(i int64) {
	// Write the index of union type, assuming that the 0-index is the null value
	e.writeByte(0x02)
	v := (uint64(i) << 1) ^ uint64(i>>63)

	e.encodeInt(v)
}

// To be used internally for writing without the union type index. Used by e.g. WriteString that needs to write the length of the string.
func (e *Encoder) writeLong(i int64) {
	v := (uint64(i) << 1) ^ uint64(i>>63)

	e.encodeInt(v)
}

// WriteString writes the binary avro encoding of the given string.
func (e *Encoder) WriteString(s string) {
	// Write the index of union type, assuming that the 0-index is the null value
	e.writeByte(0x02)

	// Write the length of the string as an Avro-encoded integer
	e.writeLong(int64(len(s)))

	// Write the bytes of the string
	e.out = append(e.out, s...)
}

// WriteFloat writes a Float to the Writer.
func (e *Encoder) WriteFloat(f float32) {
	// Write the index of union type, assuming that the 0-index is the null value
	e.writeByte(0x02)

	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, math.Float32bits(f))

	e.out = append(e.out, b...)
}

// WriteDouble writes a Double to the Writer.
func (e *Encoder) WriteDouble(f float64) {
	// Write the index of union type, assuming that the 0-index is the null value
	e.writeByte(0x02)

	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, math.Float64bits(f))

	e.out = append(e.out, b...)
}

// WriteBytes writes the binary avro encoding of the given bytes.
func (e *Encoder) WriteBytes(b []byte) {
	// Write the index of union type, assuming that the 0-index is the null value
	e.writeByte(0x02)

	// Write the length of the bytes as an Avro-encoded integer
	e.writeLong(int64(len(b)))

	// Write the bytes
	e.out = append(e.out, b...)
}

// WriteBytes writes the binary avro encoding of the given bytes.
func (e *Encoder) WriteArrayLen(i int64) {
	// Write the index of union type, assuming that the 0-index is the null value
	e.writeByte(0x02)

	e.writeLong(i)
}

// WriteBytes writes the binary avro encoding of the given bytes.
func (e *Encoder) WriteArrayEnd() {
	e.writeByte(0x00)
}

// WriteName writes the binary avro encoding of the given string. to be used for writing keys in a map.
func (e *Encoder) WriteName(s string) {
	// Write the length of the string as an Avro-encoded integer
	e.writeLong(int64(len(s)))

	// Write the bytes of the string
	e.out = append(e.out, s...)
}

// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package timestamppb contains generated types for google/protobuf/timestamp.proto.
//
// The Timestamp message represents a timestamp,
// an instant in time since the Unix epoch (January 1st, 1970).

// timestamppb.Value implements https://pkg.go.dev/database/sql/driver#Valuer.Value
// timestamppb.Scan implements https://pkg.go.dev/database/sql#Scanner.Scan
package timestamppb

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"
)

func (t *Timestamp) UnmarshalJSON(b []byte) error {
	if string(b) == "null" || string(b) == "" || string(b) == "\"\"" {
		t = &Timestamp{Seconds: 0, Nanos: 0}
		return nil
	}

	var tim time.Time
	if err := tim.UnmarshalJSON(b); err != nil {
		// TODO check and see if b is a string and if so, try to parse it as a string
		//
		return err
	}

	*t = *New(tim)
	return nil
}

func (t *Timestamp) MarshalJSON() ([]byte, error) {
	return t.AsTime().MarshalJSON()
}

func (t *Timestamp) EncodeSpanner() (interface{}, error) {
	// If our timestamp is nil, return nil and no error.
	if t == nil {
		return sql.NullTime{}, nil
	}

	return t.AsTime(), nil
}

func (t *Timestamp) Scan(src interface{}) error {
	// If our scan value is nil, set timestamp to nil and return.
	if t == nil {
		t = nil
		return nil
	}

	switch src := src.(type) {
	case nil:
		t = nil
		return nil
	case time.Time:
		*t = *New(src)
		return nil
	case string:
		t1, err := time.Parse(time.RFC3339Nano, src)
		if err != nil {
			return fmt.Errorf("error parsing timestamp data: %w", err)
		}

		*t = *New(t1)
		return nil
	}

	return fmt.Errorf("error converting timestamp data")
}

func (t *Timestamp) Value() (driver.Value, error) {
	// If our timestamp is nil, return nil and no error.
	if t == nil {
		return sql.NullTime{}, nil
	}

	return t.AsTime(), nil
}

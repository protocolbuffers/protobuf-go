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
	"database/sql/driver"
	"fmt"
	"time"

	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

func (t *Timestamp) Value() (driver.Value, error) {
	if t == nil {
		return nil, protoimpl.X.NewError("invalid nil Timestamp")
	}

	return t.AsTime(), nil
}

func (t *Timestamp) Scan(src interface{}) error {
	if t == nil {
		return protoimpl.X.NewError("invalid nil Timestamp")
	}

	switch src := src.(type) {
	case nil:
		return protoimpl.X.NewError("invalid nil Timestamp")
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

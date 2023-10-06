// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package timestamppb_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"database/sql"
	"database/sql/driver"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/internal/detrand"
	"google.golang.org/protobuf/testing/protocmp"
	tspb "google.golang.org/protobuf/types/known/timestamppb"
)

func init() {
	detrand.Disable()
}

const (
	minTimestamp = -62135596800  // Seconds between 1970-01-01T00:00:00Z and 0001-01-01T00:00:00Z, inclusive
	maxTimestamp = +253402300799 // Seconds between 1970-01-01T00:00:00Z and 9999-12-31T23:59:59Z, inclusive
)

func TestToTimestamp(t *testing.T) {
	tests := []struct {
		in   time.Time
		want *tspb.Timestamp
	}{
		{in: time.Time{}, want: &tspb.Timestamp{Seconds: -62135596800, Nanos: 0}},
		{in: time.Unix(0, 0), want: &tspb.Timestamp{Seconds: 0, Nanos: 0}},
		{in: time.Unix(math.MinInt64, 0), want: &tspb.Timestamp{Seconds: math.MinInt64, Nanos: 0}},
		{in: time.Unix(math.MaxInt64, 1e9-1), want: &tspb.Timestamp{Seconds: math.MaxInt64, Nanos: 1e9 - 1}},
		{in: time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC), want: &tspb.Timestamp{Seconds: minTimestamp, Nanos: 0}},
		{in: time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond), want: &tspb.Timestamp{Seconds: minTimestamp - 1, Nanos: 1e9 - 1}},
		{in: time.Date(9999, 12, 31, 23, 59, 59, 1e9-1, time.UTC), want: &tspb.Timestamp{Seconds: maxTimestamp, Nanos: 1e9 - 1}},
		{in: time.Date(9999, 12, 31, 23, 59, 59, 1e9-1, time.UTC).Add(+time.Nanosecond), want: &tspb.Timestamp{Seconds: maxTimestamp + 1}},
		{in: time.Date(1961, 1, 26, 0, 0, 0, 0, time.UTC), want: &tspb.Timestamp{Seconds: -281836800, Nanos: 0}},
		{in: time.Date(2011, 1, 26, 0, 0, 0, 0, time.UTC), want: &tspb.Timestamp{Seconds: 1296000000, Nanos: 0}},
		{in: time.Date(2011, 1, 26, 3, 25, 45, 940483, time.UTC), want: &tspb.Timestamp{Seconds: 1296012345, Nanos: 940483}},
	}

	for _, tt := range tests {
		got := tspb.New(tt.in)
		if diff := cmp.Diff(tt.want, got, protocmp.Transform()); diff != "" {
			t.Errorf("New(%v) mismatch (-want +got):\n%s", tt.in, diff)
		}
	}
}

func TestFromTimestamp(t *testing.T) {
	tests := []struct {
		in       *tspb.Timestamp
		wantTime time.Time
		wantErr  error
	}{
		{in: nil, wantTime: time.Unix(0, 0), wantErr: textError("invalid nil Timestamp")},
		{in: new(tspb.Timestamp), wantTime: time.Unix(0, 0)},
		{in: &tspb.Timestamp{Seconds: -62135596800, Nanos: 0}, wantTime: time.Time{}},
		{in: &tspb.Timestamp{Seconds: -1, Nanos: -1}, wantTime: time.Unix(-1, -1), wantErr: textError("timestamp (seconds:-1 nanos:-1) has out-of-range nanos")},
		{in: &tspb.Timestamp{Seconds: -1, Nanos: 0}, wantTime: time.Unix(-1, 0)},
		{in: &tspb.Timestamp{Seconds: -1, Nanos: +1}, wantTime: time.Unix(-1, +1)},
		{in: &tspb.Timestamp{Seconds: 0, Nanos: -1}, wantTime: time.Unix(0, -1), wantErr: textError("timestamp (nanos:-1) has out-of-range nanos")},
		{in: &tspb.Timestamp{Seconds: 0, Nanos: 0}, wantTime: time.Unix(0, 0)},
		{in: &tspb.Timestamp{Seconds: 0, Nanos: +1}, wantTime: time.Unix(0, +1)},
		{in: &tspb.Timestamp{Seconds: +1, Nanos: -1}, wantTime: time.Unix(+1, -1), wantErr: textError("timestamp (seconds:1 nanos:-1) has out-of-range nanos")},
		{in: &tspb.Timestamp{Seconds: +1, Nanos: 0}, wantTime: time.Unix(+1, 0)},
		{in: &tspb.Timestamp{Seconds: +1, Nanos: +1}, wantTime: time.Unix(+1, +1)},
		{in: &tspb.Timestamp{Seconds: -9876543210, Nanos: -1098765432}, wantTime: time.Unix(-9876543210, -1098765432), wantErr: textError("timestamp (seconds:-9876543210 nanos:-1098765432) has out-of-range nanos")},
		{in: &tspb.Timestamp{Seconds: +9876543210, Nanos: -1098765432}, wantTime: time.Unix(+9876543210, -1098765432), wantErr: textError("timestamp (seconds:9876543210 nanos:-1098765432) has out-of-range nanos")},
		{in: &tspb.Timestamp{Seconds: -9876543210, Nanos: +1098765432}, wantTime: time.Unix(-9876543210, +1098765432), wantErr: textError("timestamp (seconds:-9876543210 nanos:1098765432) has out-of-range nanos")},
		{in: &tspb.Timestamp{Seconds: +9876543210, Nanos: +1098765432}, wantTime: time.Unix(+9876543210, +1098765432), wantErr: textError("timestamp (seconds:9876543210 nanos:1098765432) has out-of-range nanos")},
		{in: &tspb.Timestamp{Seconds: math.MinInt64, Nanos: 0}, wantTime: time.Unix(math.MinInt64, 0), wantErr: textError("timestamp (seconds:-9223372036854775808) before 0001-01-01")},
		{in: &tspb.Timestamp{Seconds: math.MaxInt64, Nanos: 1e9 - 1}, wantTime: time.Unix(math.MaxInt64, 1e9-1), wantErr: textError("timestamp (seconds:9223372036854775807 nanos:999999999) after 9999-12-31")},
		{in: &tspb.Timestamp{Seconds: minTimestamp, Nanos: 0}, wantTime: time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)},
		{in: &tspb.Timestamp{Seconds: minTimestamp - 1, Nanos: 1e9 - 1}, wantTime: time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond), wantErr: textError("timestamp (seconds:-62135596801 nanos:999999999) before 0001-01-01")},
		{in: &tspb.Timestamp{Seconds: maxTimestamp, Nanos: 1e9 - 1}, wantTime: time.Date(9999, 12, 31, 23, 59, 59, 1e9-1, time.UTC)},
		{in: &tspb.Timestamp{Seconds: maxTimestamp + 1}, wantTime: time.Date(9999, 12, 31, 23, 59, 59, 1e9-1, time.UTC).Add(+time.Nanosecond), wantErr: textError("timestamp (seconds:253402300800) after 9999-12-31")},
		{in: &tspb.Timestamp{Seconds: -281836800, Nanos: 0}, wantTime: time.Date(1961, 1, 26, 0, 0, 0, 0, time.UTC)},
		{in: &tspb.Timestamp{Seconds: 1296000000, Nanos: 0}, wantTime: time.Date(2011, 1, 26, 0, 0, 0, 0, time.UTC)},
		{in: &tspb.Timestamp{Seconds: 1296012345, Nanos: 940483}, wantTime: time.Date(2011, 1, 26, 3, 25, 45, 940483, time.UTC)},
	}

	for _, tt := range tests {
		gotTime := tt.in.AsTime()
		if diff := cmp.Diff(tt.wantTime, gotTime); diff != "" {
			t.Errorf("AsTime(%v) mismatch (-want +got):\n%s", tt.in, diff)
		}
		gotErr := tt.in.CheckValid()
		if diff := cmp.Diff(tt.wantErr, gotErr, cmpopts.EquateErrors()); diff != "" {
			t.Errorf("CheckValid(%v) mismatch (-want +got):\n%s", tt.in, diff)
		}
	}
}

type textError string

func (e textError) Error() string     { return string(e) }
func (e textError) Is(err error) bool { return err != nil && strings.Contains(err.Error(), e.Error()) }

func TestTimestampValue(t *testing.T) {
	tests := []struct {
		in        *tspb.Timestamp
		wantValue driver.Value
		wantErr   error
	}{
		{in: nil, wantValue: sql.NullTime{}},
		{in: new(tspb.Timestamp), wantValue: new(tspb.Timestamp).AsTime()},
		{in: &tspb.Timestamp{Seconds: -62135596800, Nanos: 0}, wantValue: (&tspb.Timestamp{Seconds: -62135596800, Nanos: 0}).AsTime()},
		{in: &tspb.Timestamp{Seconds: -1, Nanos: -1}, wantValue: (&tspb.Timestamp{Seconds: -1, Nanos: -1}).AsTime()},
		{in: &tspb.Timestamp{Seconds: -1, Nanos: 0}, wantValue: time.Unix(-1, 0).UTC()},
		{in: &tspb.Timestamp{Seconds: -1, Nanos: +1}, wantValue: time.Unix(-1, +1).UTC()},
		{in: &tspb.Timestamp{Seconds: 0, Nanos: 0}, wantValue: time.Unix(0, 0).UTC()},
		{in: &tspb.Timestamp{Seconds: 0, Nanos: +1}, wantValue: time.Unix(0, +1).UTC()},
		{in: &tspb.Timestamp{Seconds: +1, Nanos: 0}, wantValue: time.Unix(+1, 0).UTC()},
		{in: &tspb.Timestamp{Seconds: +1, Nanos: +1}, wantValue: time.Unix(+1, +1).UTC()},
		{in: &tspb.Timestamp{Seconds: minTimestamp, Nanos: 0}, wantValue: time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)},
		{in: &tspb.Timestamp{Seconds: maxTimestamp, Nanos: 1e9 - 1}, wantValue: time.Date(9999, 12, 31, 23, 59, 59, 1e9-1, time.UTC)},
		{in: &tspb.Timestamp{Seconds: -281836800, Nanos: 0}, wantValue: time.Date(1961, 1, 26, 0, 0, 0, 0, time.UTC)},
		{in: &tspb.Timestamp{Seconds: 1296000000, Nanos: 0}, wantValue: time.Date(2011, 1, 26, 0, 0, 0, 0, time.UTC)},
		{in: &tspb.Timestamp{Seconds: 1296012345, Nanos: 940483}, wantValue: time.Date(2011, 1, 26, 3, 25, 45, 940483, time.UTC)},
	}

	for _, tt := range tests {
		v, gotErr := tt.in.Value()

		if gotErr == nil {
			if diff := cmp.Diff(tt.wantValue, v); diff != "" {
				t.Errorf("Value(%v) mismatch (-want +got):\n%s", tt.in, diff)
			}
		}

		if diff := cmp.Diff(tt.wantErr, gotErr, cmpopts.EquateErrors()); diff != "" {
			t.Errorf("CheckValid(%v) mismatch (-want +got):\n%s", tt.in, diff)
		}
	}
}

func TestTimestampScan(t *testing.T) {
	tests := []struct {
		in      *tspb.Timestamp
		scan    interface{}
		want    driver.Value
		wantErr error
	}{
		{in: nil, want: sql.NullTime{}, scan: nil},
		{in: &tspb.Timestamp{}, want: &tspb.Timestamp{}, scan: new(tspb.Timestamp).AsTime()},
		{in: &tspb.Timestamp{}, want: &tspb.Timestamp{Seconds: -2, Nanos: 999999999}, scan: time.Unix(-1, -1)},
		{in: &tspb.Timestamp{}, want: &tspb.Timestamp{Seconds: -1, Nanos: 0}, scan: time.Unix(-1, 0)},
		{in: &tspb.Timestamp{}, want: &tspb.Timestamp{Seconds: -1, Nanos: +1}, scan: time.Unix(-1, +1)},
		{in: &tspb.Timestamp{}, want: &tspb.Timestamp{Seconds: 0, Nanos: 0}, scan: time.Unix(0, 0)},
		{in: &tspb.Timestamp{}, want: &tspb.Timestamp{Seconds: 0, Nanos: +1}, scan: time.Unix(0, +1)},
		{in: &tspb.Timestamp{}, want: &tspb.Timestamp{Seconds: +1, Nanos: 0}, scan: time.Unix(+1, 0)},
		{in: &tspb.Timestamp{}, want: &tspb.Timestamp{Seconds: +1, Nanos: +1}, scan: time.Unix(+1, +1)},
		{in: &tspb.Timestamp{}, want: &tspb.Timestamp{Seconds: minTimestamp, Nanos: 0}, scan: time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)},
		{in: &tspb.Timestamp{}, want: &tspb.Timestamp{Seconds: maxTimestamp, Nanos: 1e9 - 1}, scan: time.Date(9999, 12, 31, 23, 59, 59, 1e9-1, time.UTC)},
		{in: &tspb.Timestamp{}, want: &tspb.Timestamp{Seconds: -281836800, Nanos: 0}, scan: time.Date(1961, 1, 26, 0, 0, 0, 0, time.UTC)},
		{in: &tspb.Timestamp{}, want: &tspb.Timestamp{Seconds: 1296000000, Nanos: 0}, scan: time.Date(2011, 1, 26, 0, 0, 0, 0, time.UTC)},
		{in: &tspb.Timestamp{}, want: &tspb.Timestamp{Seconds: 1296012345, Nanos: 940483}, scan: time.Unix(1296012345, 940483)},
		{in: &tspb.Timestamp{}, want: &tspb.Timestamp{Seconds: 1296012345, Nanos: 940483}, scan: time.Unix(1296012345, 940483).Format(time.RFC3339Nano)},
		{in: &tspb.Timestamp{}, want: &tspb.Timestamp{Seconds: 1296012345, Nanos: 940483}, scan: "invalid time string", wantErr: textError("error parsing timestamp data: parsing time \"invalid time string\" as \"2006-01-02T15:04:05.999999999Z07:00\": cannot parse \"invalid time string\" as \"2006\"")},
		{in: &tspb.Timestamp{}, want: &tspb.Timestamp{Seconds: 1296012345, Nanos: 940483}, scan: int32(123), wantErr: textError("error converting timestamp data")},
	}

	for _, tt := range tests {
		gotErr := tt.in.Scan(tt.scan)

		if gotErr == nil {
			switch ret := tt.want.(type) {
			case *tspb.Timestamp:
				if diff := cmp.Diff(tt.in, ret, protocmp.Transform()); diff != "" {
					t.Errorf("Value(%v) mismatch (-want +got):\n%s", tt.in, diff)
				}
			case sql.NullTime:
				if diff := cmp.Diff(sql.NullTime{}, ret, protocmp.Transform()); diff != "" {
					t.Errorf("Value(%v) mismatch (-want +got):\n%s", tt.in, diff)
				}
			}
		}

		if diff := cmp.Diff(tt.wantErr, gotErr, cmpopts.EquateErrors()); diff != "" {
			t.Errorf("CheckValid(%v) mismatch (-want +got):\n%s", tt.in, diff)
		}
	}
}

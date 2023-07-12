package protofif

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

var (
	ErrorInvalidTimestamp = errors.New("invalid timestamp")
	ErrorInvalidDuration  = errors.New("invalid duration")
)

func Now() *Timestamp {
	t := time.Now()
	return &Timestamp{
		Seconds: t.Unix(),
		Nanos:   int32(t.Nanosecond()),
	}
}

func NewTimestampValue(t time.Time) Timestamp {
	return Timestamp{
		Seconds: t.Unix(),
		Nanos:   int32(t.Nanosecond()),
	}
}

func NewTimestamp(t *time.Time) *Timestamp {
	if t == nil {
		return nil
	}

	return &Timestamp{
		Seconds: t.Unix(),
		Nanos:   int32(t.Nanosecond()),
	}
}

func (t *Timestamp) AsTime() *time.Time {
	if t == nil {
		return nil
	}
	ts := time.Unix(t.Seconds, int64(t.Nanos))
	return &ts
}

func (t *Timestamp) AsTimeValue() time.Time {
	if t == nil {
		return time.Time{}
	}
	return time.Unix(t.Seconds, int64(t.Nanos))
}

func (ts *Timestamp) UnmarshalJSON(b []byte) error {
	timeUnmarshaled := time.Time{}
	err := json.Unmarshal(b, &timeUnmarshaled)
	if err != nil {
		return err
	}
	*ts = NewTimestampValue(timeUnmarshaled)
	return nil
}

func (ts *Timestamp) MarshalJSON() ([]byte, error) {
	if ts == nil {
		return nil, nil
	}
	return json.Marshal(ts.AsTime())
}

func (ts *Timestamp) MarshalBSONValue() (bsontype.Type, []byte, error) {
	if ts == nil {
		return bson.TypeNull, nil, nil
	}
	return bson.MarshalValue(ts.AsTime())
}

func (ts *Timestamp) UnmarshalBSONValue(btype bsontype.Type, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if btype != bson.TypeDateTime {
		return fmt.Errorf("invalid timestamp type %s expected datetime", btype.String())
	}
	var timeVal time.Time
	err := bson.UnmarshalValue(btype, data, &timeVal)
	if err != nil {
		return err
	}
	*ts = NewTimestampValue(timeVal)
	return nil
}

// Same thing for time.Duration
func NewDurationValue(d time.Duration) Duration {
	return Duration{
		Nanoseconds: d.Nanoseconds(),
	}
}

func NewDuration(d time.Duration) *Duration {
	return &Duration{
		Nanoseconds: d.Nanoseconds(),
	}
}

func (d *Duration) AsDuration() *time.Duration {
	if d == nil {
		return nil
	}
	td := time.Duration(d.Nanoseconds)
	return &td
}

func (d *Duration) AsDurationValue() time.Duration {
	if d == nil {
		return time.Duration(0)
	}
	return time.Duration(d.Nanoseconds)
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var int64Val int64
	err := json.Unmarshal(b, &int64Val)
	if err != nil {
		return err
	}
	*d = Duration{Nanoseconds: int64Val}
	return nil
}

func (d *Duration) MarshalJSON() ([]byte, error) {
	if d == nil {
		return []byte(`""`), nil
	}
	return json.Marshal(d.Nanoseconds)
}

func (d *Duration) MarshalBSONValue() (bsontype.Type, []byte, error) {
	if d == nil {
		return bson.TypeNull, nil, nil
	}
	return bson.MarshalValue(d.Nanoseconds)
}

func (d *Duration) UnmarshalBSONValue(btype bsontype.Type, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var int64Val int64
	err := bson.UnmarshalValue(btype, data, &int64Val)
	if err != nil {
		return err
	}

	*d = Duration{Nanoseconds: int64Val}
	return nil
}

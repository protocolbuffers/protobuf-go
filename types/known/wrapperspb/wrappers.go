package wrapperspb

import (
	"database/sql"
	"encoding/json"
	"fmt"

	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

func (i *UInt32Value) EncodeSpanner() (interface{}, error) {
	if i == nil {
		return sql.NullInt64{}, nil
	}

	return sql.NullInt64{
		Int64: int64(i.Value),
		Valid: true,
	}, nil
}

func (t *UInt32Value) Scan(src interface{}) error {
	if t == nil {
		t = nil
		return nil
	}

	switch src := src.(type) {
	case nil:
		t = nil
		return nil
	case int64:
		t.Value = uint32(src)
		return nil
	}

	return fmt.Errorf("error converting timestamp data")
}

func (t *UInt32Value) UnmarshalJSON(data []byte) error {
	if t == nil {
		return protoimpl.X.NewError("invalid nil IncidentReaction")
	}
	var value uint32
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	t.Value = value
	return nil
}

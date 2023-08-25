package structpb

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

// -------------------------------------------------------------- //
// THIS IS CUSTOM CODE TO HANDLE BSON MARSHALING AND UNMARSHALING //
// -------------------------------------------------------------- //

// When converting an int64 or uint64 to a NumberValue, numeric precision loss
// is possible since they are stored as a float64.
func NewBsonValue(t bsontype.Type, v []byte) (*Value, error) {
	switch t {
	case bson.TypeNull:
		return NewNullValue(), nil
	case bson.TypeBoolean:
		// Byte to boolean
		var val bool
		err := bson.UnmarshalValue(t, v, &val)
		if err != nil {
			return nil, protoimpl.X.NewError("invalid boolean: %v | error: %v", v, err)
		}
		return NewBoolValue(val), nil
	case bson.TypeDouble, bson.TypeInt32, bson.TypeInt64:
		//Byte to float64
		var val float64
		err := bson.UnmarshalValue(t, v, &val)
		if err != nil {
			return nil, protoimpl.X.NewError("invalid number: %v | error: %v", v, err)
		}
		return NewNumberValue(val), nil
	case bson.TypeString:
		// Bson byte to string
		var val string
		err := bson.UnmarshalValue(t, v, &val)
		if err != nil {
			return nil, protoimpl.X.NewError("invalid string: %v | error: %v", v, err)
		}
		return NewStringValue(val), nil
	case bson.TypeArray:
		// Bson byte to list
		var val []interface{}
		err := bson.UnmarshalValue(t, v, &val)
		if err != nil {
			return nil, protoimpl.X.NewError("invalid list: %v | error: %v", v, err)
		}
		v2, err := NewList(val)
		if err != nil {
			return nil, err
		}
		return NewListValue(v2), nil
	case bson.TypeEmbeddedDocument:
		// Bson byte to struct
		var val map[string]interface{}
		err := bson.UnmarshalValue(t, v, &val)
		if err != nil {
			return nil, protoimpl.X.NewError("invalid struct: %v | error: %v", v, err)
		}
		v2, err := NewStruct(val)
		if err != nil {
			return nil, err
		}
		return NewStructValue(v2), nil
	default:
		return nil, protoimpl.X.NewError("invalid type: %v", t)
	}
}

func (x *Value) MarshalBSONValue() (bsontype.Type, []byte, error) {
	if x == nil {
		return bson.TypeNull, nil, nil
	}
	return bson.MarshalValue(x.AsInterface())
}

func (x *Value) UnmarshalBSONValue(t bsontype.Type, b []byte) error {
	x2, err := NewBsonValue(t, b)
	if err != nil {
		return err
	}
	*x = *x2
	return nil
}

// -------------------------------------------------------------- //

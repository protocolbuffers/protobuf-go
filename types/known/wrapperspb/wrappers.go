package wrapperspb

func (t *UInt32Value) EncodeSpanner() (interface{}, error) {
	if t == nil {
		return nil, nil
	}
	return int(t.Value), nil
}

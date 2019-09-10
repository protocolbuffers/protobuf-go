// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package impl

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"google.golang.org/protobuf/internal/descopts"
	ptag "google.golang.org/protobuf/internal/encoding/tag"
	"google.golang.org/protobuf/internal/filedesc"
	"google.golang.org/protobuf/internal/strs"
	"google.golang.org/protobuf/reflect/protoreflect"
	pref "google.golang.org/protobuf/reflect/protoreflect"
)

// legacyWrapMessage wraps v as a protoreflect.ProtoMessage,
// where v must be a *struct kind and not implement the v2 API already.
func legacyWrapMessage(v reflect.Value) pref.ProtoMessage {
	mt := legacyLoadMessageInfo(v.Type(), "")
	return mt.MessageOf(v.Interface()).Interface()
}

var legacyMessageTypeCache sync.Map // map[reflect.Type]*MessageInfo

// legacyLoadMessageInfo dynamically loads a *MessageInfo for t,
// where t must be a *struct kind and not implement the v2 API already.
// The provided name is used if it cannot be determined from the message.
func legacyLoadMessageInfo(t reflect.Type, name pref.FullName) *MessageInfo {
	// Fast-path: check if a MessageInfo is cached for this concrete type.
	if mt, ok := legacyMessageTypeCache.Load(t); ok {
		return mt.(*MessageInfo)
	}

	// Slow-path: derive message descriptor and initialize MessageInfo.
	mi := &MessageInfo{
		Desc:          legacyLoadMessageDesc(t, name),
		GoReflectType: t,
	}
	if mi, ok := legacyMessageTypeCache.LoadOrStore(t, mi); ok {
		return mi.(*MessageInfo)
	}
	return mi
}

var legacyMessageDescCache sync.Map // map[reflect.Type]protoreflect.MessageDescriptor

// LegacyLoadMessageDesc returns an MessageDescriptor derived from the Go type,
// which must be a *struct kind and not implement the v2 API already.
//
// This is exported for testing purposes.
func LegacyLoadMessageDesc(t reflect.Type) pref.MessageDescriptor {
	return legacyLoadMessageDesc(t, "")
}
func legacyLoadMessageDesc(t reflect.Type, name pref.FullName) pref.MessageDescriptor {
	// Fast-path: check if a MessageDescriptor is cached for this concrete type.
	if mi, ok := legacyMessageDescCache.Load(t); ok {
		return mi.(pref.MessageDescriptor)
	}

	// Slow-path: initialize MessageDescriptor from the raw descriptor.
	mv := reflect.New(t.Elem()).Interface()
	if _, ok := mv.(pref.ProtoMessage); ok {
		panic(fmt.Sprintf("%v already implements proto.Message", t))
	}
	mdV1, ok := mv.(messageV1)
	if !ok {
		return aberrantLoadMessageDesc(t, name)
	}
	b, idxs := mdV1.Descriptor()

	md := legacyLoadFileDesc(b).Messages().Get(idxs[0])
	for _, i := range idxs[1:] {
		md = md.Messages().Get(i)
	}
	if name != "" && md.FullName() != name {
		panic(fmt.Sprintf("mismatching message name: got %v, want %v", md.FullName(), name))
	}
	if md, ok := legacyMessageDescCache.LoadOrStore(t, md); ok {
		return md.(protoreflect.MessageDescriptor)
	}
	return md
}

var (
	aberrantMessageDescLock  sync.Mutex
	aberrantMessageDescCache map[reflect.Type]protoreflect.MessageDescriptor
)

// aberrantLoadMessageDesc returns an EnumDescriptor derived from the Go type,
// which must not implement protoreflect.ProtoMessage or messageV1.
//
// This is a best-effort derivation of the message descriptor using the protobuf
// tags on the struct fields.
func aberrantLoadMessageDesc(t reflect.Type, name pref.FullName) pref.MessageDescriptor {
	aberrantMessageDescLock.Lock()
	defer aberrantMessageDescLock.Unlock()
	if aberrantMessageDescCache == nil {
		aberrantMessageDescCache = make(map[reflect.Type]protoreflect.MessageDescriptor)
	}
	return aberrantLoadMessageDescReentrant(t, name)
}
func aberrantLoadMessageDescReentrant(t reflect.Type, name pref.FullName) pref.MessageDescriptor {
	// Fast-path: check if an MessageDescriptor is cached for this concrete type.
	if md, ok := aberrantMessageDescCache[t]; ok {
		return md
	}

	// Slow-path: construct a descriptor from the Go struct type (best-effort).
	// Cache the MessageDescriptor early on so that we can resolve internal
	// cyclic references.
	md := &filedesc.Message{L2: new(filedesc.MessageL2)}
	md.L0.FullName = aberrantDeriveMessageName(t.Elem(), name)
	md.L0.ParentFile = filedesc.SurrogateProto2
	aberrantMessageDescCache[t] = md

	// Try to determine if the message is using proto3 by checking scalars.
	for i := 0; i < t.Elem().NumField(); i++ {
		f := t.Elem().Field(i)
		if tag := f.Tag.Get("protobuf"); tag != "" {
			switch f.Type.Kind() {
			case reflect.Bool, reflect.Int32, reflect.Int64, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.String:
				md.L0.ParentFile = filedesc.SurrogateProto3
			}
			for _, s := range strings.Split(tag, ",") {
				if s == "proto3" {
					md.L0.ParentFile = filedesc.SurrogateProto3
				}
			}
		}
	}

	// Obtain a list of oneof wrapper types.
	var oneofWrappers []reflect.Type
	if fn, ok := t.MethodByName("XXX_OneofFuncs"); ok {
		vs := fn.Func.Call([]reflect.Value{reflect.Zero(fn.Type.In(0))})[3]
		for _, v := range vs.Interface().([]interface{}) {
			oneofWrappers = append(oneofWrappers, reflect.TypeOf(v))
		}
	}
	if fn, ok := t.MethodByName("XXX_OneofWrappers"); ok {
		vs := fn.Func.Call([]reflect.Value{reflect.Zero(fn.Type.In(0))})[0]
		for _, v := range vs.Interface().([]interface{}) {
			oneofWrappers = append(oneofWrappers, reflect.TypeOf(v))
		}
	}

	// Obtain a list of the extension ranges.
	if fn, ok := t.MethodByName("ExtensionRangeArray"); ok {
		vs := fn.Func.Call([]reflect.Value{reflect.Zero(fn.Type.In(0))})[0]
		for i := 0; i < vs.Len(); i++ {
			v := vs.Index(i)
			md.L2.ExtensionRanges.List = append(md.L2.ExtensionRanges.List, [2]pref.FieldNumber{
				pref.FieldNumber(v.FieldByName("Start").Int()),
				pref.FieldNumber(v.FieldByName("End").Int() + 1),
			})
			md.L2.ExtensionRangeOptions = append(md.L2.ExtensionRangeOptions, nil)
		}
	}

	// Derive the message fields by inspecting the struct fields.
	for i := 0; i < t.Elem().NumField(); i++ {
		f := t.Elem().Field(i)
		if tag := f.Tag.Get("protobuf"); tag != "" {
			tagKey := f.Tag.Get("protobuf_key")
			tagVal := f.Tag.Get("protobuf_val")
			aberrantAppendField(md, f.Type, tag, tagKey, tagVal)
		}
		if tag := f.Tag.Get("protobuf_oneof"); tag != "" {
			n := len(md.L2.Oneofs.List)
			md.L2.Oneofs.List = append(md.L2.Oneofs.List, filedesc.Oneof{})
			od := &md.L2.Oneofs.List[n]
			od.L0.FullName = md.FullName().Append(pref.Name(tag))
			od.L0.ParentFile = md.L0.ParentFile
			od.L0.Parent = md
			od.L0.Index = n

			for _, t := range oneofWrappers {
				if t.Implements(f.Type) {
					f := t.Elem().Field(0)
					if tag := f.Tag.Get("protobuf"); tag != "" {
						aberrantAppendField(md, f.Type, tag, "", "")
						fd := &md.L2.Fields.List[len(md.L2.Fields.List)-1]
						fd.L1.ContainingOneof = od
						od.L1.Fields.List = append(od.L1.Fields.List, fd)
					}
				}
			}
		}
	}

	// TODO: Use custom Marshal/Unmarshal methods for the fast-path?

	return md
}

func aberrantDeriveMessageName(t reflect.Type, name pref.FullName) pref.FullName {
	if name.IsValid() {
		return name
	}
	func() {
		defer func() { recover() }() // swallow possible nil panics
		if m, ok := reflect.New(t).Interface().(interface{ XXX_MessageName() string }); ok {
			name = pref.FullName(m.XXX_MessageName())
		}
	}()
	if name.IsValid() {
		return name
	}
	return aberrantDeriveFullName(t)
}

func aberrantAppendField(md *filedesc.Message, goType reflect.Type, tag, tagKey, tagVal string) {
	t := goType
	isOptional := t.Kind() == reflect.Ptr && t.Elem().Kind() != reflect.Struct
	isRepeated := t.Kind() == reflect.Slice && t.Elem().Kind() != reflect.Uint8
	if isOptional || isRepeated {
		t = t.Elem()
	}
	fd := ptag.Unmarshal(tag, t, placeholderEnumValues{}).(*filedesc.Field)

	// Append field descriptor to the message.
	n := len(md.L2.Fields.List)
	md.L2.Fields.List = append(md.L2.Fields.List, *fd)
	fd = &md.L2.Fields.List[n]
	fd.L0.FullName = md.FullName().Append(fd.Name())
	fd.L0.ParentFile = md.L0.ParentFile
	fd.L0.Parent = md
	fd.L0.Index = n

	if fd.L1.IsWeak || fd.L1.HasPacked {
		fd.L1.Options = func() pref.ProtoMessage {
			opts := descopts.Field.ProtoReflect().New()
			if fd.L1.IsWeak {
				opts.Set(opts.Descriptor().Fields().ByName("weak"), protoreflect.ValueOf(true))
			}
			if fd.L1.HasPacked {
				opts.Set(opts.Descriptor().Fields().ByName("packed"), protoreflect.ValueOf(fd.L1.IsPacked))
			}
			return opts.Interface()
		}
	}

	// Populate Enum and Message.
	if fd.Enum() == nil && fd.Kind() == pref.EnumKind {
		switch v := reflect.Zero(t).Interface().(type) {
		case pref.Enum:
			fd.L1.Enum = v.Descriptor()
		default:
			fd.L1.Enum = LegacyLoadEnumDesc(t)
		}
	}
	if fd.Message() == nil && (fd.Kind() == pref.MessageKind || fd.Kind() == pref.GroupKind) {
		switch v := reflect.Zero(t).Interface().(type) {
		case pref.ProtoMessage:
			fd.L1.Message = v.ProtoReflect().Descriptor()
		case messageV1:
			fd.L1.Message = LegacyLoadMessageDesc(t)
		default:
			if t.Kind() == reflect.Map {
				n := len(md.L1.Messages.List)
				md.L1.Messages.List = append(md.L1.Messages.List, filedesc.Message{L2: new(filedesc.MessageL2)})
				md2 := &md.L1.Messages.List[n]
				md2.L0.FullName = md.FullName().Append(pref.Name(strs.MapEntryName(string(fd.Name()))))
				md2.L0.ParentFile = md.L0.ParentFile
				md2.L0.Parent = md
				md2.L0.Index = n

				md2.L2.IsMapEntry = true
				md2.L2.Options = func() pref.ProtoMessage {
					opts := descopts.Message.ProtoReflect().New()
					opts.Set(opts.Descriptor().Fields().ByName("map_entry"), protoreflect.ValueOf(true))
					return opts.Interface()
				}

				aberrantAppendField(md2, t.Key(), tagKey, "", "")
				aberrantAppendField(md2, t.Elem(), tagVal, "", "")

				fd.L1.Message = md2
				break
			}
			fd.L1.Message = aberrantLoadMessageDescReentrant(t, "")
		}
	}
}

type placeholderEnumValues struct {
	protoreflect.EnumValueDescriptors
}

func (placeholderEnumValues) ByNumber(n pref.EnumNumber) pref.EnumValueDescriptor {
	return filedesc.PlaceholderEnumValue(pref.FullName(fmt.Sprintf("UNKNOWN_%d", n)))
}

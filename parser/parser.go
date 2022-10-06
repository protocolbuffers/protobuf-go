package parser

import (
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/infiniteloopcloud/protoc-gen-go-types/compiler/protogen"
	"github.com/infiniteloopcloud/protoc-gen-go-types/types/descriptorpb"
	"github.com/infiniteloopcloud/protoc-gen-go-types/types/pluginpb"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jinzhu/copier"
)

func Parse(filenames ...string) (*protogen.Plugin, error) {
	p := protoparse.Parser{}
	fdp, err := p.ParseFilesButDoNotLink(filenames...)
	if err != nil {
		return nil, err
	}

	var (
		major     = int32(3)
		minor     = int32(21)
		patch     = int32(5)
		suffix    = ""
		parameter = "M=.;proto"
	)
	req := &pluginpb.CodeGeneratorRequest{
		ProtoFile:      convert(fdp),
		FileToGenerate: filenames,
		CompilerVersion: &pluginpb.Version{
			Major:  &major,
			Minor:  &minor,
			Patch:  &patch,
			Suffix: &suffix,
		},
		Parameter: &parameter,
	}
	opts := protogen.Options{}
	return opts.New(req)
}

func convert(fdp []*descriptor.FileDescriptorProto) []*descriptorpb.FileDescriptorProto {
	var f []*descriptorpb.FileDescriptorProto
	copier.Copy(f, fdp)
	for _, sfdp := range fdp {
		data := &descriptorpb.FileDescriptorProto{}
		copier.Copy(data, sfdp)
		if data.Options == nil {
			data.Options = &descriptorpb.FileOptions{}
		}
		parameter := ".;proto"
		data.Options.GoPackage = &parameter
		f = append(f, data)
	}
	return f
}

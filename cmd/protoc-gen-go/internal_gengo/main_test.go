package internal_gengo

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/infiniteloopcloud/protoc-gen-go-types/parser"
)

func TestGenerateFile(t *testing.T) {
	t.Setenv("TYPE_OVERRIDE", "true")
	gen, err := parser.Parse("google/protobuf/descriptor.proto", "./test_data/config.proto", "./test_data/test.proto")
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range gen.Files {
		if f.Generate {
			content, err := GenerateFile(gen, f).Content()
			if err != nil {
				t.Fatal(err)
			}
			f, err := os.Create("./test_data/" + f.GeneratedFilenamePrefix + ".pb.go")
			if err != nil {
				t.Fatal(err)
			}
			io.Copy(f, bytes.NewReader(content))
		}
	}
}

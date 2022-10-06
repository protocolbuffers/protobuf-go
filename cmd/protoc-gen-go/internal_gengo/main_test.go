package internal_gengo

import (
	"testing"

	"github.com/infiniteloopcloud/protoc-gen-go-types/parser"
)

func TestGenerateFile(t *testing.T) {
	gen, err := parser.Parse("./test_data/test.proto")
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range gen.Files {
		if f.Generate {
			GenerateFile(gen, f)
		}
	}
}

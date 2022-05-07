package internal_gengo

import (
	"fmt"
	"google.golang.org/protobuf/compiler/protogen"
	"regexp"
	"testing"
)

func TestCommentTagsReg(t *testing.T) {
	re, err := regexp.Compile(`(\s?)@go_tags\(` + "(`.*`)" + `\)\s`)
	if err != nil {
		t.Error(err)
		return
	}

	str := "   @go_tags(`json:\"name\"`) abc"
	matched := re.FindStringSubmatch(str)
	fmt.Println(len(matched), matched)
}

func TestParseGoTagsFromTailingComment(t *testing.T) {
	str := "  @go_tags(`json:\"name,omitempty\"`) abc"
	tags, newTailing := ParseGoTagsFromTailingComment(protogen.Comments(str))
	for key, value := range tags {
		fmt.Println(key, value)
	}
	fmt.Println(newTailing)
}

func TestAppendGoTagsFromTailingComment(t *testing.T) {
	tags := structTags{
		{"protobuf", "abc"},
		{"json", "efg"},
		{"protobuf_key", "string"},
		{"protobuf_val", "string"},
	}
	str := "  @go_tags(`json:\"name,omitempty\" bson:\"name\"`) abc"
	newTags, newTailing := AppendGoTagsFromTailingComment(tags, protogen.Comments(str))
	fmt.Println(newTailing)
	fmt.Println(newTags)
}

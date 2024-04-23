// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// Added by Hao Luo <haozzzzzzzz@gmail.com>
// Source:  https://github.com/hacksomecn/protobuf-go.git
// Branch: feature/tags

// Package internal_gengo Support add custom struct field tags
// Protoc parse field tags from field's tailing comment, declare extra tags like:
// message Example {
// ...
// string name = 1; // @go_tags(`bson:"name" yaml:"name"`) FORM 1 support comment tail
// ...
// }
//
// FORM 1: Go tags regexp: `(\s?)@go_tags\(` + "(`.*`)" + `\)\s`
package internal_gengo

import (
	"google.golang.org/protobuf/compiler/protogen"
	"log"
	"regexp"
	"strings"
)

var tailingGoTagsExcludeKeys = map[string]bool{
	"protobuf":     true,
	"protobuf_key": true,
	"protobuf_val": true,
}

var commentGoTagsRe *regexp.Regexp

func init() {
	var err error
	commentGoTagsRe, err = regexp.Compile(`(\s?)@go_tags\(` + "(`.*`)" + `\)\s`)
	if err != nil {
		log.Fatalf("compile comment go tags regexp failed. %s", err)
		return
	}
}

// AppendGoTagsFromFieldComment append extra tags parsed from tailing comment
// tag with same name will be replaced except protobuf tags like "protobuf"、"protobuf_key"、"protobuf_val"
func AppendGoTagsFromFieldComment(
	existsTags structTags,
	tailComment protogen.Comments,
) (
	newTags structTags,
	newTailing protogen.Comments,
) {
	newTags = existsTags
	newTailing = tailComment

	tagsMap := map[string]string{} // key -> value
	seqKeys := make([]string, 0)
	for _, existTags := range existsTags {
		key := existTags[0]
		value := existTags[1]
		tagsMap[key] = value
		seqKeys = append(seqKeys, key)
	}

	tailTags, newTailing := ParseGoTagsFromTailingComment(tailComment)
	for _, tailTag := range tailTags {
		key := tailTag.Key
		value := tailTag.Value
		if tailingGoTagsExcludeKeys[key] {
			continue
		}

		_, exists := tagsMap[key]
		if !exists { // keep sequence
			seqKeys = append(seqKeys, key)
		}

		tagsMap[key] = value
	}

	newTags = make([][2]string, 0)
	for _, key := range seqKeys {
		tag := tagsMap[key]
		newTags = append(newTags, [2]string{key, tag})
	}

	return
}

type GoTag struct {
	Key   string
	Value string
}

// ParseGoTagsFromTailingComment parse go tags from comment
func ParseGoTagsFromTailingComment(tailing protogen.Comments) (
	tags []GoTag,
	newTailing protogen.Comments,
) {
	newTailing = tailing

	matched := commentGoTagsRe.FindStringSubmatch(string(tailing))
	if len(matched) != 3 {
		return
	}

	strMatched := matched[0]
	strStart := matched[1]
	strTagsReplacer := strings.Replace(strMatched, strStart, "", 1)
	newTailing = protogen.Comments(strings.Replace(string(tailing), strTagsReplacer, "", 1))

	strTags := matched[2]
	strTags = strings.Trim(strTags, "`")

	strPairs := strings.Split(strTags, " ")
	for _, pair := range strPairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		separateIndex := strings.Index(pair, ":")
		if separateIndex < 0 || separateIndex == len(pair)-1 {
			continue
		}

		key := pair[:separateIndex]
		value := pair[separateIndex+1:]
		value = strings.Trim(value, "\"")

		tags = append(tags, GoTag{
			Key:   key,
			Value: value,
		})
	}

	return
}

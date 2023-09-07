package main

import (
	"slices"
	"testing"
)

func TestCheckHTML(t *testing.T) {
	var tests = []struct {
		input string
		want  []string
	}{
		{"", []string{}},
		{"simple text", []string{}},
		{"<start>", []string{errStartWithoutEnd("start")}},
		{"<start></end>", []string{errStartEndMismatch("start", "end")}},
		{"</end>", []string{errEndWithoutStart("end")}},
		{"</end><start>", []string{errEndWithoutStart("end"), errStartWithoutEnd("start")}},
		{"<a><label></a>", []string{errStartEndMismatch("label", "a"), errStartWithoutEnd("a")}},
		{"<a><label>some label</label>some text<tag></a>", []string{errStartEndMismatch("tag", "a"), errStartWithoutEnd("a")}},
		{"<img src='foo'>image here</img>", []string{}},
		{"<img src=\"img\">image<br/>here</img>", []string{}},
		{"<img src=\"img\">image<br>here</img>", []string{errStartEndMismatch("br", "img"), errStartWithoutEnd("img")}},
		{"text1<tag>text2</img>text3</tag>text4", []string{errStartEndMismatch("tag", "img"), errEndWithoutStart("tag")}},
	}
	for _, test := range tests {
		if got := checkHTML(test.input); !slices.Equal(got, test.want) {
			t.Errorf("want: %q, got: %q", test.want, got)
		}
	}
}

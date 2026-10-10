package lgr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecretSet_Add(t *testing.T) {
	tbl := []struct {
		name string
		adds [][]string
		want []string
	}{
		{name: "nothing added", adds: nil, want: nil},
		{name: "longest first", adds: [][]string{{"abc", "abcdef", "zz"}}, want: []string{"abcdef", "abc", "zz"}},
		{name: "same length in byte order", adds: [][]string{{"bbb", "aaa"}}, want: []string{"aaa", "bbb"}},
		{name: "duplicates in one call", adds: [][]string{{"abc", "abc"}}, want: []string{"abc"}},
		{name: "duplicates across calls", adds: [][]string{{"abc"}, {"abc", "abcd"}}, want: []string{"abcd", "abc"}},
		{name: "empty and whitespace ignored", adds: [][]string{{"", " ", "\t\n", "abc"}}, want: []string{"abc"}},
		{name: "accepted value not trimmed", adds: [][]string{{" abc "}}, want: []string{" abc "}},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			s := &secretSet{}
			for _, vals := range tt.adds {
				s.add(vals...)
			}
			assert.Equal(t, tt.want, secretStrings(s.list()))
		})
	}
}

func TestSecretSet_Remove(t *testing.T) {
	tbl := []struct {
		name    string
		initial []string
		remove  []string
		want    []string
	}{
		{name: "removes present", initial: []string{"abc", "abcdef"}, remove: []string{"abc"}, want: []string{"abcdef"}},
		{name: "added twice removed once", initial: []string{"abc", "abc"}, remove: []string{"abc"}, want: nil},
		{name: "absent value is no-op", initial: []string{"abc"}, remove: []string{"xyz", ""}, want: []string{"abc"}},
		{name: "removes several", initial: []string{"a1", "b22", "c333"}, remove: []string{"c333", "a1"}, want: []string{"b22"}},
		{name: "remove from empty", initial: nil, remove: []string{"abc"}, want: nil},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			s := &secretSet{}
			s.add(tt.initial...)
			s.remove(tt.remove...)
			assert.Equal(t, tt.want, secretStrings(s.list()))
		})
	}
}

func TestSecretSet_WritesKeepPublishedListIntact(t *testing.T) {
	s := &secretSet{}
	s.add("abc", "zz")
	before := s.list()
	s.add("abcdef")
	afterAdd := s.list()
	s.remove("abc")
	assert.Equal(t, []string{"abc", "zz"}, secretStrings(before))
	assert.Equal(t, []string{"abcdef", "abc", "zz"}, secretStrings(afterAdd))
	assert.Equal(t, []string{"abcdef", "zz"}, secretStrings(s.list()))
}

func TestSecretSet_NilList(t *testing.T) {
	var s *secretSet
	assert.Empty(t, s.list())
}

func TestSecretList_Hide(t *testing.T) {
	tbl := []struct {
		name       string
		own, other []string
		in, want   string
	}{
		{name: "no secrets", in: "some abc text", want: "some abc text"},
		{name: "own only", own: []string{"abc"}, in: "some abc text abc", want: "some ****** text ******"},
		{name: "other only", other: []string{"abc"}, in: "some abc text", want: "some ****** text"},
		{name: "both lists", own: []string{"abc"}, other: []string{"xyz"}, in: "abc and xyz", want: "****** and ******"},
		{name: "longer value in other list", own: []string{"abc"}, other: []string{"abcdef"}, in: "abcdef abc", want: "****** ******"},
		{name: "longer value in own list", own: []string{"abcdef"}, other: []string{"abc"}, in: "abcdef abc", want: "****** ******"},
		{name: "same value in both", own: []string{"abc"}, other: []string{"abc"}, in: "x abc y", want: "x ****** y"},
		{name: "interleaved lengths", own: []string{"abcdefgh", "abc"}, other: []string{"abcdef", "a"}, in: "abcdefgh abcdef abc a",
			want: "****** ****** ****** ******"},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			own, other := &secretSet{}, &secretSet{}
			own.add(tt.own...)
			other.add(tt.other...)
			assert.Equal(t, tt.want, string(own.list().hide([]byte(tt.in), other.list())))
		})
	}
}

func secretStrings(sl secretList) []string {
	if len(sl) == 0 {
		return nil
	}
	res := make([]string, 0, len(sl))
	for _, v := range sl {
		res = append(res, string(v))
	}
	return res
}

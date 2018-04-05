package main

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/finkf/corpus"
)

func TestThawNumber(t *testing.T) {
	tests := []uint64{
		23418, 1234567890,
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("%d", tc), func(t *testing.T) {
			buf := &bytes.Buffer{}
			if err := freeze(tc, buf); err != nil {
				t.Fatalf("got errror: %v", err)
			}
			var got uint64
			if err := thaw(&got, buf); err != nil {
				t.Fatalf("got error: %v", err)
			}
			if got != tc {
				t.Fatalf("expected %d; got %d", tc, got)
			}
		})
	}
}

func TestThawTrigrams(t *testing.T) {
	tests := []struct {
		name string
		data []*corpus.Trigrams
	}{
		{"single trigram", []*corpus.Trigrams{
			new(corpus.Trigrams).Add("abc", "def", "ghi"),
		}},
		{"multiple trigrams", []*corpus.Trigrams{
			new(corpus.Trigrams).Add("abc", "def", "ghi"),
			new(corpus.Trigrams).Add("abc", "ghi", "def"),
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			for _, x := range tc.data {
				if err := freeze(x, buf); err != nil {
					t.Fatalf("got errror: %v", err)
				}
			}
			var got corpus.Trigrams
			if err := thaw(&got, buf); err != nil {
				t.Fatalf("got error: %v", err)
			}
			var want corpus.Trigrams
			for _, x := range tc.data {
				(&want).Append(x)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("expected %v; got %v", tc, got)
			}
		})
	}
}

func TestThawCharTrigrams(t *testing.T) {
	tests := []struct {
		name string
		data []*corpus.CharTrigrams
	}{
		{"single trigram", []*corpus.CharTrigrams{
			new(corpus.CharTrigrams).Add("one two three"),
		}},
		{"multiple trigrams", []*corpus.CharTrigrams{
			new(corpus.CharTrigrams).Add("one two three"),
			new(corpus.CharTrigrams).Add("fourfivesize"),
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			for _, x := range tc.data {
				if err := freeze(x, buf); err != nil {
					t.Fatalf("got errror: %v", err)
				}
			}
			var got corpus.CharTrigrams
			if err := thaw(&got, buf); err != nil {
				t.Fatalf("got error: %v", err)
			}
			var want corpus.CharTrigrams
			for _, x := range tc.data {
				(&want).Append(x)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("expected %v; got %v", tc, got)
			}
		})
	}
}

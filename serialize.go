package main

import (
	"encoding/gob"
	"io"
	"log"
	"os"

	"github.com/finkf/corpus"
)

func thawFromFile(data interface{}, path string) error {
	log.Printf("thawing from %s", path)
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	return thaw(data, in)
}

func thaw(data interface{}, r io.Reader) error {
	switch t := data.(type) {
	case *corpus.Trigrams:
		return thawTrigrams(t, r)
	case *corpus.CharTrigrams:
		return thawCharTrigrams(t, r)
	default:
		return gob.NewDecoder(r).Decode(data)
	}
}

func thawTrigrams(t *corpus.Trigrams, r io.Reader) error {
	for {
		var tmp corpus.Trigrams
		err := gob.NewDecoder(r).Decode(&tmp)
		if err == io.EOF { // done
			return nil
		}
		if err != nil {
			return err
		}
		t.Append(&tmp)
	}
}

func thawCharTrigrams(t *corpus.CharTrigrams, r io.Reader) error {
	for {
		var tmp corpus.CharTrigrams
		err := gob.NewDecoder(r).Decode(&tmp)
		if err == io.EOF { // done
			return nil
		}
		if err != nil {
			return err
		}
		t.Append(&tmp)
	}
}

func freezeToFile(data interface{}, path string) (e2 error) {
	log.Printf("freezing to %s", path)
	out, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() { e2 = closeHelper(e2, out.Close()) }()
	return freeze(data, out)
}

func freeze(data interface{}, w io.Writer) error {
	return gob.NewEncoder(w).Encode(data)
}

func closeHelper(err, closeErr error) error {
	if err != nil {
		return err
	}
	return closeErr
}

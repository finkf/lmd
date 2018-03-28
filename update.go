package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/finkf/corpus"
	"github.com/spf13/cobra"
)

var (
	update = &cobra.Command{
		Use:   "update",
		Long:  "update character and token 3-grams",
		Short: "update the language models",
		Run:   doUpdate,
	}
	tickets        chan struct{}
	char3gramsChan chan *corpus.Char3Grams
	trigramsChan   chan *corpus.Trigrams
)

func doUpdate(cmd *cobra.Command, args []string) {
	ensure(os.MkdirAll(datadir, os.ModePerm))
	var wg sync.WaitGroup
	char3gramsChan = make(chan *corpus.Char3Grams)
	trigramsChan = make(chan *corpus.Trigrams)
	tickets = make(chan struct{}, 500)
	for i, arg := range args {
		wg.Add(1)
		go func(i int, arg string) {
			defer wg.Done()
			updateFile(i, arg)
		}(i+1, arg)
	}
	var wg2 sync.WaitGroup
	wg2.Add(1)
	go gather3grams(&wg2)
	var wg3 sync.WaitGroup
	wg3.Add(1)
	go gatherTrigrams(&wg3)
	wg.Wait()
	close(char3gramsChan)
	wg2.Wait()
	close(trigramsChan)
	wg3.Wait()
	close(tickets)
}

func gather3grams(wg *sync.WaitGroup) {
	defer wg.Done()
	ngrams := corpus.NewChar3Grams()
	for m := range char3gramsChan {
		ngrams.Add3Grams(m)
	}
	ensure(writeLM(ngrams, "char3grams.json.gz"))
}

func gatherTrigrams(wg *sync.WaitGroup) {
	defer wg.Done()
	m := make(map[rune]*corpus.Trigrams)
	var i int
	var total uint64
	for t := range trigramsChan {
		if i >= 10 {
			i = 0
			for r, t := range m {
				ensure(updateTrigrams(lmFileNameFromRune(r), t))
				delete(m, r)
			}
		}
		t.Each(func(str string, b *corpus.Bigrams) {
			r := runeFromFirstString(str)
			if _, ok := m[r]; !ok {
				m[r] = new(corpus.Trigrams)
			}
			m[r].AddBigrams(str, b)
		})
		i++
		total += t.Total()
	}
	for r, t := range m {
		ensure(updateTrigrams(lmFileNameFromRune(r), t))
	}
	ensure(writeLM(total, "total.json.gz"))
}

func updateTrigrams(path string, t *corpus.Trigrams) error {
	var other corpus.Trigrams
	if readLM(&other, path) == nil { // we do not care about errors
		t.AddTrigrams(&other)
	}
	return writeLM(t, path)
}

func readLM(data interface{}, path string) error {
	path = filepath.Join(datadir, path)
	log.Printf("reading %s", path)
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	zip, err := gzip.NewReader(in)
	if err != nil {
		return err
	}
	defer func() { _ = zip.Close() }()
	return json.NewDecoder(zip).Decode(data)
}

func writeLM(data interface{}, path string) (e2 error) {
	path = filepath.Join(datadir, path)
	log.Printf("writing %s", path)
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { e2 = closeHelper(e2, out.Close()) }()
	zip := gzip.NewWriter(out)
	defer func() { e2 = closeHelper(e2, zip.Close()) }()
	return json.NewEncoder(zip).Encode(data)
}

func closeHelper(err, closeErr error) error {
	if err != nil {
		return err
	}
	return closeErr
}

func updateFile(i int, uri string) {
	tickets <- struct{}{}
	log.Printf("%d: uri: %s", i, uri)
	is, err := os.Open(uri)
	ensure(err)
	m := corpus.NewChar3Grams()
	var tokens []string
	corpus.DTAReadTokensAndClose(is, func(t corpus.Token) {
		if t.Type() != corpus.Word {
			return
		}
		l := strings.ToLower(string(t))
		m.AddAll(l)
		tokens = append(tokens, l)
	})
	char3gramsChan <- m
	trigramsChan <- new(corpus.Trigrams).Add(tokens...)
	_ = <-tickets
}

func runeFromFirstString(str string) rune {
	r, _ := utf8.DecodeRuneInString(str)
	if r == utf8.RuneError {
		r = '_'
	}
	if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
		r = '_'
	}
	return r
}

func lmFileNameFromRune(r rune) string {
	return fmt.Sprintf("%cNgrams.json.gz", r)
}

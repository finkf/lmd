package main

import (
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/finkf/corpus"
	"github.com/spf13/cobra"
)

type wpair struct {
	f string
	t interface{}
}

type totaler interface {
	Total() uint64
}

var (
	update = &cobra.Command{
		Use:   "update",
		Long:  "update character and token 3-grams",
		Short: "update the language models",
		Run:   doUpdate,
	}
	tickets        chan struct{}
	char3gramsChan chan *corpus.CharTrigrams
	trigramsChan   chan *corpus.Trigrams
	writeChan      chan wpair
	blocksize      int
	workers        int
)

func init() {
	update.Flags().IntVarP(&blocksize, "block-size", "b",
		10*1000, "set the block size to write out trigrams")
	update.Flags().IntVarP(&workers, "workers", "w",
		runtime.NumCPU(), "set number of parallel workers")
}

func doUpdate(cmd *cobra.Command, args []string) {
	ensure(os.MkdirAll(datadir, os.ModePerm))
	char3gramsChan = make(chan *corpus.CharTrigrams, 2*workers)
	trigramsChan = make(chan *corpus.Trigrams, 2*workers)
	writeChan = make(chan wpair, 2*workers)
	tickets = make(chan struct{}, workers)

	var wg sync.WaitGroup
	go doArgs(args)
	wg.Add(1)
	go gather3grams(&wg)
	wg.Add(1)
	go gatherTrigrams(&wg)
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go writeTrigrams(&wg)
	}
	wg.Wait()
}

func write(t totaler, f string, bs uint64) bool {
	if t.Total() <= bs {
		return false
	}
	log.Printf("writing total: %d (%d)", t.Total(), bs)
	writeChan <- wpair{t: t, f: f}
	return true
}

func writeTrigrams(wg *sync.WaitGroup) {
	defer wg.Done()
	for t := range writeChan {
		ensure(writeLM(t.t, t.f))
	}
}

func gather3grams(wg *sync.WaitGroup) {
	defer wg.Done()
	ngrams := new(corpus.CharTrigrams)
	for m := range char3gramsChan {
		ngrams.Append(m)
		if write(ngrams, "char3grams.gob", uint64(blocksize)) {
			ngrams = new(corpus.CharTrigrams)
		}
	}
	write(ngrams, "char3grams.gob", 0)
}

func gatherTrigrams(wg *sync.WaitGroup) {
	defer wg.Done()
	m := make([]*corpus.Trigrams, hash)
	var total uint64
	for t := range trigramsChan {
		t.Each(func(str string, b *corpus.Bigrams) {
			h := hashFromString(str)
			if m[h] == nil {
				m[h] = new(corpus.Trigrams)
			}
			m[h].AppendBigrams(str, b)
		})
		total += t.Total()
		for h, t := range m {
			if write(t, pathFromHash(h), uint64(blocksize)) {
				m[h] = nil
			}
		}
	}
	for h, t := range m {
		write(t, pathFromHash(h), 0)
	}
	ensure(writeLM(total, "total.gob"))
	close(writeChan)
}

func readLM(data interface{}, path string) error {
	path = filepath.Join(datadir, path)
	return thawFromFile(data, path)
}

func writeLM(data interface{}, path string) error {
	path = filepath.Join(datadir, path)
	return freezeToFile(data, path)
}

func updateFile(wg *sync.WaitGroup, i int, uri string) {
	defer wg.Done()
	log.Printf("%d: uri: %s", i, uri)
	is, err := os.Open(uri)
	ensure(err)
	m := new(corpus.CharTrigrams)
	var tokens []string
	ensure(corpus.DTAReadTokensAndClose(is, func(t corpus.Token) {
		if t.Type() != corpus.Word {
			return
		}
		l := strings.ToLower(string(t))
		m.Add(l)
		tokens = append(tokens, l)
	}))
	log.Printf("sending %d char 3-grams", m.Total())
	char3gramsChan <- m
	log.Printf("sending %d tokens", len(tokens))
	trigramsChan <- new(corpus.Trigrams).Add(tokens...)
	<-tickets
}

func doArgs(args []string) {
	var wg sync.WaitGroup
	for i, arg := range args {
		tickets <- struct{}{}
		wg.Add(1)
		go updateFile(&wg, i+1, arg)
	}
	wg.Wait()
	close(char3gramsChan)
	close(trigramsChan)
	close(tickets)
}

func hashFromString(str string) int {
	h := fnv.New32()
	_, _ = h.Write([]byte(str))
	return int(h.Sum32()) % hash
}

func pathFromHash(h int) string {
	return fmt.Sprintf("%04xNgrams.gob", h)
}

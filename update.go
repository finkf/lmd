package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"

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
	tickets chan struct{}
)

func doUpdate(cmd *cobra.Command, args []string) {
	var wg sync.WaitGroup
	charmaps := make(chan *corpus.Char3Grams)
	tickets = make(chan struct{}, 500)
	doArgs(args, func(i int, uri string) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			updateFile(i, uri, charmaps)
		}()
	})
	var wg2 sync.WaitGroup
	wg2.Add(1)
	go func() {
		defer wg2.Done()
		ngrams := corpus.NewChar3Grams()
		for m := range charmaps {
			ngrams.Add3Grams(m)
		}
		ensure(json.NewEncoder(os.Stdout).Encode(ngrams))
	}()
	wg.Wait()
	close(charmaps)
	wg2.Wait()
}

func updateFile(i int, uri string, maps chan<- *corpus.Char3Grams) {
	tickets <- struct{}{}
	log.Printf("%d: uri: %s", i, uri)
	is, err := os.Open(uri)
	ensure(err)
	m := corpus.NewChar3Grams()
	corpus.DTAReadTokensAndClose(is, func(t corpus.Token) {
		m.AddAll(string(strings.ToLower(string(t))))
	})
	maps <- m
	_ = <-tickets
}

func doArgs(args []string, f func(int, string)) {
	for i, arg := range args {
		f(i+1, arg)
	}
}

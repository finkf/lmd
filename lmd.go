package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"regexp"

	"github.com/finkf/corpus"
	"github.com/finkf/lmd/api"
	"github.com/finkf/qparams"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	lmd = &cobra.Command{
		Use:   "lmd",
		Long:  "Language Model Daemon",
		Short: "Language Model Daemon",
		Run:   doLMD,
	}
	char3grams *corpus.CharTrigrams
	host       string
	datadir    string
	total      uint64
	hash       int
)

func init() {
	lmd.AddCommand(update)
	lmd.PersistentFlags().StringVarP(&datadir, "dir", "d", "data", "set data dir")
	lmd.PersistentFlags().IntVarP(&hash, "hash", "s", 128, "set hash size")
	lmd.Flags().StringVarP(&host, "host", "H", "localhost:8080", "set host address")
}

func execute() error {
	return lmd.Execute()
}

func doLMD(cmd *cobra.Command, args []string) {
	char3grams = new(corpus.CharTrigrams)
	ensure(readLM(char3grams, "char3grams.gob"))
	ensure(readLM(&total, "total.gob"))
	http.HandleFunc(api.CharTrigramURL, handleCharTrigrams)
	http.HandleFunc(api.TrigramURL, handleNGrams)
	log.Printf("starting server on %s", host)
	http.ListenAndServe(host, nil)
}

func handleCharTrigrams(w http.ResponseWriter, r *http.Request) {
	log.Printf("handling %s", r.URL)
	var h handle
	var q api.CharTrigramRequest
	h.decodeQuery(&q, r)
	var x api.CharTrigramResponse
	h.exec(func() (int, error) {
		var err error
		x, err = searchCharTrigrams(q)
		if err != nil {
			return http.StatusBadRequest, err
		}
		return http.StatusOK, nil
	})
	h.writeJSON(x, w)
	status, err := h.status()
	if err != nil {
		http.Error(w, err.Error(), status)
	}
	log.Printf("handled %s: %v [%d]", r.URL, err, status)
}

func handleNGrams(w http.ResponseWriter, r *http.Request) {
	log.Printf("handling %s", r.URL)
	var h handle
	var q api.TrigramRequest
	h.decodeQuery(&q, r)
	var x api.TrigramResponse
	h.exec(func() (int, error) {
		var err error
		x, err = searchNGrams(q)
		if err != nil {
			return http.StatusInternalServerError, err
		}
		return http.StatusOK, nil
	})
	h.writeJSON(x, w)
	status, err := h.status()
	if err != nil {
		http.Error(w, err.Error(), status)
	}
	log.Printf("handled %s: %v [%d]", r.URL, err, status)
}

func searchCharTrigrams(r api.CharTrigramRequest) (api.CharTrigramResponse, error) {
	res := api.CharTrigramResponse{
		CharTrigramRequest: r,
		Total:              char3grams.Total(),
	}
	if !r.Regex {
		res.Matches = append(res.Matches, api.CharTrigramMatch{NGram: r.Q, Count: char3grams.Get(r.Q)})
		return res, nil
	}
	re, err := regexp.Compile(r.Q)
	if err != nil {
		return res, errors.Wrapf(err, "invalid regex: %s", r.Q)
	}
	char3grams.Each(func(k string, v uint64) {
		if re.MatchString(k) {
			res.Matches = append(res.Matches, api.CharTrigramMatch{NGram: k, Count: v})
		}
	})
	return res, nil
}

func searchNGrams(r api.TrigramRequest) (api.TrigramResponse, error) {
	res := api.TrigramResponse{
		TrigramRequest: r,
		Total:          total,
	}
	if len(r.F) == 0 {
		return res, nil
	}
	path := pathFromHash(hashFromString(r.F))
	var t corpus.Trigrams
	if readLM(&t, path) != nil {
		return res, nil
	}
	if len(r.S) > 0 {
		if len(r.T) > 0 {
			res.Matches = t.Get(r.F).Get(r.S).Get(r.T)
			return res, nil
		}
		res.Matches = t.Get(r.F).Get(r.S)
		return res, nil
	}
	res.Matches = t.Get(r.F)
	return res, nil
}

type handle struct {
	err  error
	stat int
}

func (h *handle) exec(f func() (int, error)) {
	if h.err != nil {
		return
	}
	status, err := f()
	h.err = err
	h.stat = status
}

func (h *handle) decodeQuery(q interface{}, r *http.Request) {
	h.exec(func() (int, error) {
		if err := qparams.Decode(r.URL.Query(), q); err != nil {
			return http.StatusBadRequest, err
		}
		return http.StatusOK, nil
	})
}

func (h *handle) writeJSON(x interface{}, w http.ResponseWriter) {
	h.exec(func() (int, error) {
		buf := &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(x); err != nil {
			return http.StatusInternalServerError, err
		}
		if _, err := w.Write(buf.Bytes()); err != nil {
			return http.StatusInternalServerError, err
		}
		return http.StatusOK, nil
	})
}

func (h *handle) status() (int, error) {
	return h.stat, h.err
}

func ensure(err error) {
	if err != nil {
		panic(err)
	}
}

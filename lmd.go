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
	char3grams *corpus.Char3Grams
	host       string
	datadir    string
)

func init() {
	lmd.AddCommand(update)
	lmd.PersistentFlags().StringVarP(&datadir, "dir", "d", "data", "set data dir")
	lmd.Flags().StringVarP(&host, "host", "t", "localhost:8080", "set host address")
}

func execute() error {
	return lmd.Execute()
}

func doLMD(cmd *cobra.Command, args []string) {
	char3grams = corpus.NewChar3Grams()
	ensure(readLM(char3grams, "char3grams.json.gz"))
	http.HandleFunc("/char3grams", handleChar3Grams)
	http.HandleFunc("/ngrams", handleNGrams)
	log.Printf("starting server on %s", host)
	http.ListenAndServe(host, nil)
}

func handleChar3Grams(w http.ResponseWriter, r *http.Request) {
	log.Printf("handling %s", r.URL)
	var h handle
	var q api.Char3GramsRequest
	h.decodeQuery(&q, r)
	var x api.Char3GramsResponse
	h.exec(func() (int, error) {
		var err error
		x, err = searchChar3Grams(q)
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
	var q api.NGramsRequest
	h.decodeQuery(&q, r)
	var x api.NGramsResponse
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

func searchChar3Grams(r api.Char3GramsRequest) (api.Char3GramsResponse, error) {
	res := api.Char3GramsResponse{
		Char3GramsRequest: r,
		Total:             char3grams.Total(),
	}
	if !r.Regex {
		res.Matches = append(res.Matches, api.CharNGramMatch{NGram: r.Q, Count: char3grams.Get(r.Q)})
		return res, nil
	}
	re, err := regexp.Compile(r.Q)
	if err != nil {
		return res, errors.Wrapf(err, "invalid regex: %s", r.Q)
	}
	char3grams.Each(func(k string, v uint64) {
		if re.MatchString(k) {
			res.Matches = append(res.Matches, api.CharNGramMatch{NGram: k, Count: v})
		}
	})
	return res, nil
}

func searchNGrams(r api.NGramsRequest) (api.NGramsResponse, error) {
	res := api.NGramsResponse{
		NGramsRequest: r,
	}
	if len(r.F) == 0 {
		return res, nil
	}
	path := lmFileNameFromRune(runeFromFirstString(r.F))
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

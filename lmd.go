package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
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
)

func init() {
	lmd.AddCommand(update)
}

func execute() error {
	return lmd.Execute()
}

func doLMD(cmd *cobra.Command, args []string) {
	var err error
	char3grams, err = readChar3Grams("char3grams.json")
	ensure(err)
	http.HandleFunc("/char3grams", handleChar3Grams)
	log.Printf("starting server")
	http.ListenAndServe("localhost:8080", nil)
}

func handleChar3Grams(w http.ResponseWriter, r *http.Request) {
	log.Printf("handling %s", r.URL)
	var h handle
	var q api.Char3GramsRequest
	h.exec(func() (int, error) {
		if err := qparams.Decode(r.URL.Query(), &q); err != nil {
			return http.StatusBadRequest, err
		}
		return http.StatusOK, nil
	})
	var x api.Char3GramsResponse
	h.exec(func() (int, error) {
		var err error
		x, err = searchChar3Grams(q, char3grams)
		if err != nil {
			return http.StatusBadRequest, err
		}
		return http.StatusOK, nil
	})
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
	status, err := h.status()
	if err != nil {
		http.Error(w, err.Error(), status)
	}
	log.Printf("handled %s: %v [%d]", r.URL, err, status)
}

func searchChar3Grams(r api.Char3GramsRequest, char3grams *corpus.Char3Grams) (api.Char3GramsResponse, error) {
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

func readChar3Grams(path string) (*corpus.Char3Grams, error) {
	is, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot open character 3-grams")
	}
	defer func() { _ = is.Close() }()
	var char3grams corpus.Char3Grams
	if err := json.NewDecoder(is).Decode(&char3grams); err != nil {
		return nil, errors.Wrapf(err, "cannot decode character 3-grams")
	}
	return &char3grams, nil
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

func (h *handle) status() (int, error) {
	return h.stat, h.err
}

func ensure(err error) {
	if err != nil {
		panic(err)
	}
}

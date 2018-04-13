// Package api defines the interface for
// requests to the language model.
package api

// CharTrigramRequest defines the data
// for a request for character 3-grams.
type CharTrigramRequest struct {
	Q     string
	Regex bool
}

// CharTrigramResponse defines the data
// for the response of a request for character 3-grams.
type CharTrigramResponse struct {
	CharTrigramRequest
	Total   uint64
	Matches []CharTrigramMatch
}

// CharTrigramMatch defines a (string, uint64) match pair.
type CharTrigramMatch struct {
	NGram string
	Count uint64
}

// TrigramRequest defines the data
// for a request for token n-grams.
type TrigramRequest struct {
	F, S, T string
}

// TrigramResponse defines the data
// for the respones of a request for token n-grams.
type TrigramResponse struct {
	TrigramRequest
	Total   uint64
	Matches interface{}
}

// Default paths for the different api requests.
const (
	CharTrigramURL = "/chartrigram"
	TrigramURL     = "/trigram"
)

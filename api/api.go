// Package api defines the interface for
// requests to the language model.
package api

// Char3GramsRequest defines the data
// for a request for character 3-grams.
type Char3GramsRequest struct {
	Q     string
	Regex bool
}

// Char3GramsResponse defines the data
// for the response of a request for character 3-grams.
type Char3GramsResponse struct {
	Char3GramsRequest
	Total   uint64
	Matches []CharNGramMatch
}

// CharNGramMatch defines a (string, uint64) match pair.
type CharNGramMatch struct {
	NGram string
	Count uint64
}

// NGramsRequest defines the data
// for a request for token n-grams.
type NGramsRequest struct {
	F, S, T string
}

// NGramsResponse defines the data
// for the respones of a request for token n-grams.
type NGramsResponse struct {
	NGramsRequest
	Total   uint64
	Matches interface{}
}

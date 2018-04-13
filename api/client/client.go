package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/finkf/lmd/api"
	"github.com/finkf/logger"
	"github.com/finkf/qparams"
)

// Default settings for the client.
const (
	DefaultHost    = "localhost:8080"
	DefaultTimeout = 1 * time.Minute
)

// Client represents an api client with an
// underlying http.Client instance.
type Client struct {
	log    logger.Logger
	host   string
	client *http.Client
}

// Opt is a functional option.
type Opt func(*Client)

// WithHost defines a custom host configuration.
// c := client.New(client.WithHost("myhost"))
func WithHost(host string) Opt {
	return func(c *Client) {
		c.host = host
	}
}

// WithTimeout defines a custom timeout configuration
// c := client.New(client.WithTimeout(10 * time.Second))
func WithTimeout(to time.Duration) Opt {
	return func(c *Client) {
		c.client.Timeout = to
	}
}

// WithLogger sets the logger for the client to use.
// c := client.New(client.WithLogger(logger.New(...)))
func WithLogger(l logger.Logger) Opt {
	return func(c *Client) {
		c.log = l
	}
}

// New creates a new client that connects to the given host.
func New(opts ...Opt) Client {
	c := Client{
		host:   DefaultHost,
		client: &http.Client{Timeout: DefaultTimeout},
		log:    logger.Nil(),
	}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// CharTrigram issues a new request for the given char-trigrams
// to the host and returns its response.
func (c Client) CharTrigram(q string, regex bool) (api.CharTrigramResponse, error) {
	req := api.CharTrigramRequest{Q: q, Regex: regex}
	g := &getter{client: c}
	url := g.url(api.CharTrigramURL, req)
	var res api.CharTrigramResponse
	g.get(url, &res)
	return res, g.err()
}

// Trigram issues a new request for the given trigrams
// to the host and returns its response.
func (c Client) Trigram(f, s, t string) (api.TrigramResponse, error) {
	req := api.TrigramRequest{F: f, S: s, T: t}
	g := &getter{client: c}
	url := g.url(api.TrigramURL, req)
	var res api.TrigramResponse
	g.get(url, &res)
	return res, g.err()
}

type getter struct {
	_err   error // to prevent collision with err()
	client Client
}

func (g *getter) url(path string, q interface{}) string {
	if g._err != nil {
		return ""
	}
	params, err := qparams.Encode(q)
	if err != nil {
		g._err = err
		return ""
	}
	return fmt.Sprintf("%s%s%s", g.client.host, path, params)
}

func (g *getter) get(url string, r interface{}) {
	if g._err != nil {
		return
	}
	g.client.log.Debugf("request: [GET] %s")
	resp, err := g.client.client.Get(url)
	if err != nil {
		g._err = err
		return
	}
	defer func() { _ = resp.Body.Close() }()
	g._err = json.NewDecoder(resp.Body).Decode(r)
}

func (g *getter) err() error {
	return g._err
}

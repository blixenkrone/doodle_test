package znhttp

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/blixenkrone/sdk/logger"
)

// TODO: Needs an actual implementation specific to ZN and not just using stdlib
// TODO: Add wrappers for DataDog
type HTTPDoer interface {
	Do(r *http.Request) (*http.Response, error)
}

var _ HTTPDoer = HTTPClient{}

type HTTPClient struct {
	logger      logger.Logger
	client      *http.Client
	serviceName string

	verboseLogging bool
}

func NewClient(logger logger.Logger, serviceName string, opts ...Option[HTTPClientOptions]) HTTPClient {
	var options HTTPClientOptions
	for _, opt := range opts {
		opt(&options)
	}

	timeout := time.Second * 30
	if v, ok := options.Timeout(); ok {
		timeout = v
	}

	transport := http.DefaultTransport.(*http.Transport)
	stdlibClient := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	return HTTPClient{
		logger:         logger,
		client:         stdlibClient,
		serviceName:    serviceName,
		verboseLogging: options.VerboseLogging(),
	}
}

// Do implements HTTPDoer and does an HTTP request.
func (c HTTPClient) Do(r *http.Request) (*http.Response, error) {
	// The standard client's Do method respects the context associated with the request via req.WithContext().

	log := c.logger.WithContext(r.Context())

	if c.verboseLogging {
		// TODO: Implement a better way to do verbose logging
		log.Debugf("HTTP %s: %s", r.Method, r.URL.String())
	}

	resp, err := c.client.Do(r)
	if err != nil {
		if os.IsTimeout(err) {
			log.Warnf("request to %s timed out", r.URL.String())
		}
		return nil, fmt.Errorf("http client failed Do() request to %q: %w", r.URL.String(), err)
	}
	if c.verboseLogging {
		log.Debugf("HTTP %s: %s response code: %d", r.Method, r.URL.String(), resp.StatusCode)
	}

	return resp, nil
}

type Option[T any] func(*T)
type HTTPClientOptions struct {
	timeout        time.Duration
	verboseLogging bool // TODO: implement this for request body and headers, also redacting sensitive information
}

func WithTimeout(d time.Duration) Option[HTTPClientOptions] {
	return func(o *HTTPClientOptions) {
		o.timeout = d
	}
}

func (c HTTPClientOptions) Timeout() (time.Duration, bool) {
	return c.timeout, c.timeout != 0
}
func WithVerboseLogging() Option[HTTPClientOptions] {
	return func(o *HTTPClientOptions) {
		o.verboseLogging = true
	}
}

func (c HTTPClientOptions) VerboseLogging() bool {
	return c.verboseLogging
}

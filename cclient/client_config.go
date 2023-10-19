package cclient

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	fhttp "github.com/bogdanfinn/fhttp"
	tlsclient "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

//////////////////////////////////////////////////

var DefaultHTTPClientOptions = []tlsclient.HttpClientOption{
	tlsclient.WithTimeoutSeconds(10),
	tlsclient.WithClientProfile(profiles.Chrome_105),
}

var DefaultClientHeader = http.Header{
	"Accept":          {"*/*"},
	"Accept-Language": {"en-US,en;q=0.7"},
	"Cache-Control":   {"no-cache"},
	"Pragma":          {"no-cache"},
	"User-Agent":      {"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.75 Safari/537.36"},

	fhttp.HeaderOrderKey: {
		"Accept",
		"Accept-Language",
		"User-Agent",
	},
}

//////////////////////////////////////////////////

type clientConfig struct {
	logger            *slog.Logger
	httpClient        tlsclient.HttpClient
	httpClientOptions []tlsclient.HttpClientOption
	defaultHeader     http.Header
}

var NilClientConfig = errors.New("config is nil")

func validateClientConfig(cfg *clientConfig) error {
	if cfg == nil {
		return NilClientConfig
	}

	return nil
}

func buildClientFromConfig(cfg *clientConfig) (c *BasicClient, err error) {
	if cfg == nil {
		err = NilClientConfig
		return
	}

	c = &BasicClient{
		logger: cfg.logger,
	}

	if cfg.httpClient != nil {
		c.httpClient = cfg.httpClient
	} else {
		var httpClientOptions []tlsclient.HttpClientOption

		if cfg.httpClientOptions != nil {
			httpClientOptions = cfg.httpClientOptions
		} else {
			jar := tlsclient.NewCookieJar()

			httpClientOptions = append(
				DefaultHTTPClientOptions,
				tlsclient.WithCookieJar(jar),
			)
		}

		c.httpClient, err = tlsclient.NewHttpClient(
			tlsclient.NewNoopLogger(),
			httpClientOptions...,
		)
		if err != nil {
			return nil, fmt.Errorf("tlsclient.NewHttpClient: %w", err)
		}
	}

	if cfg.defaultHeader != nil {
		c.defaultHeader = cfg.defaultHeader.Clone()
	} else {
		if DefaultClientHeader != nil {
			c.defaultHeader = DefaultClientHeader.Clone()
		} else {
			c.defaultHeader = http.Header{}
		}
	}

	return
}

type ClientConfigOption func(cfg *clientConfig)

//////////////////////////////////////////////////

func WithLogger(logger *slog.Logger) ClientConfigOption {
	return func(cfg *clientConfig) {
		cfg.logger = logger
	}
}

func WithHTTPClient(httpClient tlsclient.HttpClient) ClientConfigOption {
	return func(cfg *clientConfig) {
		cfg.httpClient = httpClient
	}
}

func WithHTTPClientOptions(httpClientOptions []tlsclient.HttpClientOption) ClientConfigOption {
	return func(cfg *clientConfig) {
		cfg.httpClientOptions = httpClientOptions
	}
}

func WithDefaultHeader(defaultHeader http.Header) ClientConfigOption {
	return func(cfg *clientConfig) {
		cfg.defaultHeader = defaultHeader
	}
}

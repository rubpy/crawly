package cclient

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	fhttp "github.com/bogdanfinn/fhttp"
	tlsclient "github.com/bogdanfinn/tls-client"

	"github.com/rubpy/crawly/clog"
)

//////////////////////////////////////////////////

type Client interface {
	HTTPClient() tlsclient.HttpClient

	DefaultHeader() http.Header
	SetDefaultHeader(header http.Header)
	Logger() *slog.Logger
	SetLogger(logger *slog.Logger)
	Log(ctx context.Context, params clog.Params)

	CookieJar() fhttp.CookieJar
	SetCookieJar(jar fhttp.CookieJar)
	SetCookies(u *url.URL, cookies []*fhttp.Cookie)

	Request(ctx context.Context, method string, url string, body io.Reader, headers http.Header) (*fhttp.Response, error)
}

type BasicClient struct {
	httpClient    tlsclient.HttpClient
	logger        *slog.Logger
	defaultHeader http.Header
}

func NewClient(opts ...ClientConfigOption) (*BasicClient, error) {
	var cfg clientConfig

	for _, opt := range opts {
		opt(&cfg)
	}

	if err := validateClientConfig(&cfg); err != nil {
		return nil, err
	}

	c, err := buildClientFromConfig(&cfg)
	if err != nil {
		return nil, err
	}

	return c, nil
}

//////////////////////////////////////////////////

func (c *BasicClient) HTTPClient() tlsclient.HttpClient {
	return c.httpClient
}

func (c *BasicClient) DefaultHeader() http.Header {
	return c.defaultHeader
}

func (c *BasicClient) SetDefaultHeader(header http.Header) {
	c.defaultHeader = header
}

func (c *BasicClient) Logger() *slog.Logger {
	return c.logger
}

func (c *BasicClient) SetLogger(logger *slog.Logger) {
	c.logger = logger
}

func (c *BasicClient) Log(ctx context.Context, params clog.Params) {
	if c.logger == nil {
		return
	}

	clog.WithParams(c.logger, ctx, params)
}

func (c *BasicClient) CookieJar() fhttp.CookieJar {
	return c.httpClient.GetCookieJar()
}

func (c *BasicClient) SetCookieJar(jar fhttp.CookieJar) {
	c.httpClient.SetCookieJar(jar)
}

func (c *BasicClient) SetCookies(u *url.URL, cookies []*fhttp.Cookie) {
	c.httpClient.SetCookies(u, cookies)
}

func (c *BasicClient) Request(ctx context.Context, method string, url string, body io.Reader, headers http.Header) (*fhttp.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	} else if err := ctx.Err(); err != nil {
		return nil, err
	}

	req, err := fhttp.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header = fhttp.Header(c.defaultHeader)
	if req.Header == nil {
		req.Header = fhttp.Header{}
	}
	if headers != nil {
		for k, v := range headers {
			k = http.CanonicalHeaderKey(k)
			req.Header[k] = v
		}
	}

	lp := clog.Params{
		Message: "request",
		Level:   slog.LevelDebug,

		Values: clog.ParamGroup{
			"method": method,
			"url":    url,
		},
	}

	resp, err := c.httpClient.Do(req)
	if err == nil {
		lp.Set("status", resp.StatusCode)
	}

	lp.Err = err
	c.Log(ctx, lp)

	return resp, err
}

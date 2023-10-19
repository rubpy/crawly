package cclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	fhttp "github.com/bogdanfinn/fhttp"

	"github.com/rubpy/crawly/clog"
)

//////////////////////////////////////////////////

type APIResponse interface {
	Close() error

	Raw() *fhttp.Response
	Header() http.Header
	StatusCode() int

	Body() (body []byte, err error)
	Decode(result any) error
}

type APIClient interface {
	Client() Client
	Logger() *slog.Logger
	SetLogger(logger *slog.Logger)
	Log(ctx context.Context, params clog.Params)

	BaseURL() string
	SetBaseURL(baseURL string)
	DefaultHeader() http.Header
	SetDefaultHeader(defaultHeader http.Header)

	Request(ctx context.Context, method string, endpointURI string, urlParams URLParams, headers http.Header, bodyData interface{}) (response APIResponse, err error)
	RawRequest(ctx context.Context, method string, url string, headers http.Header, body io.Reader) (response APIResponse, err error)
}

type basicAPIClient struct {
	client Client
	logger *slog.Logger

	baseURL       string
	defaultHeader http.Header
}

var NilClient = errors.New("client is nil")

func NewAPIClient(logger *slog.Logger, client Client, baseURL string, defaultHeader http.Header) (APIClient, error) {
	return &basicAPIClient{
		client: client,
		logger: logger,

		baseURL:       baseURL,
		defaultHeader: defaultHeader,
	}, nil
}

func (api *basicAPIClient) Client() Client {
	return api.client
}

func (api *basicAPIClient) Logger() *slog.Logger {
	return api.logger
}

func (api *basicAPIClient) SetLogger(logger *slog.Logger) {
	api.logger = logger
}

func (api *basicAPIClient) Log(ctx context.Context, params clog.Params) {
	if api.logger == nil {
		return
	}

	clog.WithParams(api.logger, ctx, params)
}

func (api *basicAPIClient) BaseURL() string {
	return api.baseURL
}

func (api *basicAPIClient) SetBaseURL(baseURL string) {
	api.baseURL = strings.TrimSuffix(baseURL, "/") + "/"
}

func (api *basicAPIClient) DefaultHeader() http.Header {
	return api.defaultHeader
}

func (api *basicAPIClient) SetDefaultHeader(defaultHeader http.Header) {
	api.defaultHeader = defaultHeader
}

func (api *basicAPIClient) Request(ctx context.Context, method string, endpointURI string, urlParams URLParams, headers http.Header, data interface{}) (response APIResponse, err error) {
	url := api.baseURL + strings.TrimPrefix(endpointURI, "/")
	if urlParams != nil {
		url += "?" + urlParams.Encode()
	}

	var body io.Reader = nil
	if data != nil {
		var b []byte

		var ok bool
		if b, ok = data.([]byte); !ok {
			b, err = json.Marshal(data)
			if err != nil {
				return
			}
		}

		body = bytes.NewReader(b)
	}

	lp := clog.Params{
		Message: "request",
		Level:   slog.LevelDebug,

		Values: clog.ParamGroup{
			"method": method,
			"url":    url,
		},
	}

	response, err = api.RawRequest(ctx, method, url, headers, body)

	lp.Err = err
	api.Log(ctx, lp)

	return
}

func (api *basicAPIClient) RawRequest(ctx context.Context, method string, url string, headers http.Header, body io.Reader) (response APIResponse, err error) {
	if api.client == nil {
		return nil, NilClient
	}

	if ctx == nil {
		ctx = context.Background()
	} else {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
	}

	hdr := api.defaultHeader.Clone()
	if hdr == nil {
		hdr = http.Header{}
	}
	if headers != nil {
		for k, v := range headers {
			k = http.CanonicalHeaderKey(k)
			hdr[k] = v
		}
	}

	r, err := api.client.Request(ctx, method, url, body, hdr)
	if r != nil && r.StatusCode == http.StatusNotModified {
		if r.Body != nil {
			r.Body.Close()
		}

		err = &APIError{
			Code:   r.StatusCode,
			Header: http.Header(r.Header),
		}
		return
	}
	if err != nil {
		return
	}

	if err = checkAPIResponse(r); err != nil {
		return
	}

	response = &basicAPIResponse{
		raw: r,
	}
	return
}

type basicAPIResponse struct {
	raw *fhttp.Response

	body   []byte
	closed bool
}

func (r *basicAPIResponse) Close() error {
	if r.closed {
		return nil
	}

	err := r.raw.Body.Close()
	if err == nil {
		r.closed = true
	}

	return err
}

func (r *basicAPIResponse) Raw() *fhttp.Response {
	return r.raw
}

func (r *basicAPIResponse) Header() http.Header {
	return http.Header(r.raw.Header)
}

func (r *basicAPIResponse) StatusCode() int {
	return r.raw.StatusCode
}

func (r *basicAPIResponse) Body() (body []byte, err error) {
	if r.body != nil {
		return r.body, nil
	}

	body, err = io.ReadAll(r.raw.Body)
	if err == nil {
		r.body = body
	}

	return
}

func (r *basicAPIResponse) Decode(result any) error {
	body, err := r.Body()
	if err != nil {
		return err
	}

	if err := json.NewDecoder(bytes.NewReader(body)).Decode(result); err != nil {
		return err
	}

	return nil
}

//////////////////////////////////////////////////

type APIError struct {
	err error

	Response *APIResponse `json:"response"`
	Code     int          `json:"code"`
	Header   http.Header  `json:"header"`
	Message  string       `json:"message"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("cclient.APIClient: error %d (%s)", e.Code, e.Message)
	}

	return fmt.Sprintf("cclient.APIClient: got HTTP response code %d", e.Code)
}

func (e *APIError) Wrap(err error) {
	e.err = err
}

func (e *APIError) Unwrap() error {
	return e.err
}

func checkAPIResponse(res *fhttp.Response) error {
	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		return nil
	}

	return &APIError{
		Response: nil,
		Code:     res.StatusCode,
		Header:   http.Header(res.Header),
	}
}

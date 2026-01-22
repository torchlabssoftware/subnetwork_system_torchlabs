package helpers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type TestClient struct {
	BaseURL  string
	APIKey   string
	Client   *http.Client
	AuthType string
}

func NewAdminClient(baseURL, apiKey string) *TestClient {
	return &TestClient{
		BaseURL:  baseURL,
		APIKey:   apiKey,
		Client:   &http.Client{},
		AuthType: "admin",
	}
}

func NewWorkerClient(baseURL, apiKey string) *TestClient {
	return &TestClient{
		BaseURL:  baseURL,
		APIKey:   apiKey,
		Client:   &http.Client{},
		AuthType: "worker",
	}
}

type RequestOptions struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
}

type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

func (tc *TestClient) DoRequest(t *testing.T, opts RequestOptions) *Response {
	var bodyReader io.Reader
	if opts.Body != nil {
		bodyBytes, err := json.Marshal(opts.Body)
		require.NoError(t, err, "failed to marshal request body")
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	url := tc.BaseURL + opts.Path
	req, err := http.NewRequest(opts.Method, url, bodyReader)
	require.NoError(t, err, "failed to create request")

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "ApiKey "+tc.APIKey)

	for key, value := range opts.Headers {
		req.Header.Set(key, value)
	}

	resp, err := tc.Client.Do(req)
	require.NoError(t, err, "failed to execute request")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       body,
		Headers:    resp.Header,
	}
}

func (tc *TestClient) Get(t *testing.T, path string) *Response {
	return tc.DoRequest(t, RequestOptions{
		Method: http.MethodGet,
		Path:   path,
	})
}

func (tc *TestClient) Post(t *testing.T, path string, body interface{}) *Response {
	return tc.DoRequest(t, RequestOptions{
		Method: http.MethodPost,
		Path:   path,
		Body:   body,
	})
}

func (tc *TestClient) Put(t *testing.T, path string, body interface{}) *Response {
	return tc.DoRequest(t, RequestOptions{
		Method: http.MethodPut,
		Path:   path,
		Body:   body,
	})
}

func (tc *TestClient) Delete(t *testing.T, path string) *Response {
	return tc.DoRequest(t, RequestOptions{
		Method: http.MethodDelete,
		Path:   path,
	})
}

func (tc *TestClient) DeleteWithBody(t *testing.T, path string, body interface{}) *Response {
	return tc.DoRequest(t, RequestOptions{
		Method: http.MethodDelete,
		Path:   path,
		Body:   body,
	})
}

func (r *Response) ParseJSON(t *testing.T, target interface{}) {
	err := json.Unmarshal(r.Body, target)
	require.NoError(t, err, "failed to parse response JSON: %s", string(r.Body))
}

func (r *Response) AssertStatus(t *testing.T, expected int) {
	if r.StatusCode != expected {
		t.Errorf("expected status %d, got %d. Response body: %s", expected, r.StatusCode, string(r.Body))
	}
}

func (r *Response) String() string {
	return string(r.Body)
}

func Ptr[T any](v T) *T {
	return &v
}

func (r *Response) RequireStatus(t *testing.T, expected int) {
	require.Equal(t, expected, r.StatusCode, "unexpected status code. Response: %s", string(r.Body))
}

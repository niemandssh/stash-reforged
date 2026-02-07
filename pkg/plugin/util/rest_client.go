package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"

	"github.com/stashapp/stash/pkg/plugin/common"
)

// RESTClient provides a simple REST API client for Go plugins.
// It connects to the Stash REST API at /api/v1.
type RESTClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewRESTClient creates a REST API client connecting to the stash server
// using the provided server connection details.
func NewRESTClient(provider common.StashServerConnection) *RESTClient {
	portStr := strconv.Itoa(provider.Port)

	u, _ := url.Parse("http://" + provider.Host + ":" + portStr + "/api/v1")
	u.Scheme = provider.Scheme

	cookieJar, _ := cookiejar.New(nil)

	cookie := provider.SessionCookie
	if cookie != nil {
		cookieJar.SetCookies(u, []*http.Cookie{
			cookie,
		})
	}

	httpClient := &http.Client{
		Jar: cookieJar,
	}

	return &RESTClient{
		baseURL:    u.String(),
		httpClient: httpClient,
	}
}

// Get performs a GET request to the given path (relative to /api/v1).
func (c *RESTClient) Get(path string, result interface{}) error {
	return c.doRequest("GET", path, nil, result)
}

// Post performs a POST request with a JSON body.
func (c *RESTClient) Post(path string, body interface{}, result interface{}) error {
	return c.doRequest("POST", path, body, result)
}

// Put performs a PUT request with a JSON body.
func (c *RESTClient) Put(path string, body interface{}, result interface{}) error {
	return c.doRequest("PUT", path, body, result)
}

// Patch performs a PATCH request with a JSON body.
func (c *RESTClient) Patch(path string, body interface{}, result interface{}) error {
	return c.doRequest("PATCH", path, body, result)
}

// Delete performs a DELETE request with an optional JSON body.
func (c *RESTClient) Delete(path string, body interface{}, result interface{}) error {
	return c.doRequest("DELETE", path, body, result)
}

func (c *RESTClient) doRequest(method, path string, body interface{}, result interface{}) error {
	fullURL := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("REST API error %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshaling response: %w", err)
		}
	}

	return nil
}

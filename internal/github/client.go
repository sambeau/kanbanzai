// Package github provides GitHub API operations for PR management.
package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DefaultBaseURL is the default GitHub API base URL.
const DefaultBaseURL = "https://api.github.com"

// Client provides GitHub API operations.
type Client struct {
	token      string
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new GitHub client with the given token.
func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{},
		baseURL:    DefaultBaseURL,
	}
}

// NewClientWithHTTPClient creates a new GitHub client with a custom http.Client.
// This is useful for testing with mocked HTTP responses.
func NewClientWithHTTPClient(token string, client *http.Client) *Client {
	return &Client{
		token:      token,
		httpClient: client,
		baseURL:    DefaultBaseURL,
	}
}

// SetBaseURL sets the base URL for API requests.
// This is useful for testing with a mock server.
func (c *Client) SetBaseURL(url string) {
	c.baseURL = strings.TrimSuffix(url, "/")
}

// doRequest performs an HTTP request with authentication and error handling.
func (c *Client) doRequest(method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	return resp, nil
}

// checkResponse checks the HTTP response for errors and returns appropriate error types.
func (c *Client) checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		// Check if it's rate limiting
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			return ErrRateLimited
		}
		return ErrUnauthorized
	case http.StatusNotFound:
		return ErrRepoNotFound
	case http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		return fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, string(bodyBytes))
	}
}

// decodeResponse decodes a JSON response body into the target struct.
func (c *Client) decodeResponse(resp *http.Response, target any) error {
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// get performs a GET request and decodes the response.
func (c *Client) get(path string, target any) error {
	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return err
	}

	if err := c.checkResponse(resp); err != nil {
		return err
	}

	return c.decodeResponse(resp, target)
}

// post performs a POST request and decodes the response.
func (c *Client) post(path string, body, target any) error {
	resp, err := c.doRequest(http.MethodPost, path, body)
	if err != nil {
		return err
	}

	if err := c.checkResponse(resp); err != nil {
		return err
	}

	if target != nil {
		return c.decodeResponse(resp, target)
	}

	resp.Body.Close()
	return nil
}

// patch performs a PATCH request and decodes the response.
func (c *Client) patch(path string, body, target any) error {
	resp, err := c.doRequest(http.MethodPatch, path, body)
	if err != nil {
		return err
	}

	if err := c.checkResponse(resp); err != nil {
		return err
	}

	if target != nil {
		return c.decodeResponse(resp, target)
	}

	resp.Body.Close()
	return nil
}

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	ServerURL string
	JWT       string
	APIKey    string
}

func New(serverURL string) *Client {
	return &Client{ServerURL: strings.TrimRight(serverURL, "/")}
}

func (c *Client) do(method, path string, body io.Reader) (*http.Response, error) {
	url := c.ServerURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	} else if c.JWT != "" {
		req.Header.Set("Authorization", "Bearer "+c.JWT)
	}

	return http.DefaultClient.Do(req)
}

func (c *Client) getJSON(path string, dest interface{}) error {
	resp, err := c.do(http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s", resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(dest)
}

func (c *Client) postJSON(path string, payload, dest interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		return err
	}

	resp, err := c.do(http.MethodPost, path, &buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	if dest != nil {
		return json.NewDecoder(resp.Body).Decode(dest)
	}
	return nil
}

func (c *Client) Login(email, password string) (*LoginResult, error) {
	payload := map[string]string{
		"email":    email,
		"password": password,
	}

	var result LoginResult
	err := c.postJSON("/auth/login", payload, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) CreateBucket(name string) (*Bucket, error) {
	payload := map[string]string{"name": name}

	var result Bucket
	err := c.postJSON("/buckets", payload, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) ListBuckets() ([]Bucket, error) {
	var result []Bucket
	err := c.getJSON("/buckets", &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

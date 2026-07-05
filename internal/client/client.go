package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
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

func (c *Client) newRequest(method, path string, body io.Reader) (*http.Request, error) {
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

	return req, nil
}

func (c *Client) do(method, path string, body io.Reader) (*http.Response, error) {
	req, err := c.newRequest(method, path, body)
	if err != nil {
		return nil, err
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(body)))
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

func (c *Client) deleteReq(path string) error {
	resp, err := c.do(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s", resp.Status)
	}
	return nil
}

func (c *Client) Login(email, password string) (*LoginResult, error) {
	payload := map[string]string{"email": email, "password": password}
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

func (c *Client) UploadObject(bucket, filePath string, body io.Reader) (*Object, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(part, body)
	if err != nil {
		return nil, err
	}

	writer.Close()

	req, err := c.newRequest(
		http.MethodPut,
		"/buckets/"+bucket+"/objects/"+filepath.Base(filePath),
		&buf,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var result Object
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DownloadObject(bucket, key string) (*http.Response, error) {
	return c.do(http.MethodGet, "/buckets/"+bucket+"/objects/"+key, nil)
}

func (c *Client) DeleteObject(bucket, key string) error {
	return c.deleteReq("/buckets/" + bucket + "/objects/" + key)
}

func (c *Client) ListObjects(bucket string) ([]Object, error) {
	var result []Object
	err := c.getJSON("/buckets/"+bucket+"/objects/", &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) SignURL(bucket, key, operation, expiresIn string) (*SignResult, error) {
	payload := map[string]string{
		"operation":  operation,
		"expires_in": expiresIn,
	}
	var result SignResult
	err := c.postJSON("/buckets/"+bucket+"/objects/"+key+"/sign", payload, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

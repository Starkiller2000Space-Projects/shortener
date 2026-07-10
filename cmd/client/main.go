// Package main implements a simple HTTP client for the URL shortener.

package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// makeRequest creates an HTTP POST request with form-encoded data.
// It adds the appropriate Content-Type header.
func makeRequest(endpoint string, data url.Values) (*http.Request, error) {
	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	return request, nil
}

// HTTPClient wraps http.Client with a custom transport for connection pooling.
type HTTPClient struct {
	*http.Client
}

// NewHTTPClient creates a new HTTP client with connection pooling and timeouts.
func NewHTTPClient() *HTTPClient {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.RetryWaitMin = 1 * time.Second
	retryClient.RetryWaitMax = 5 * time.Second

	transport := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	retryClient.HTTPClient.Transport = transport
	retryClient.HTTPClient.Timeout = 30 * time.Second

	stdClient := retryClient.StandardClient()

	return &HTTPClient{
		Client: stdClient,
	}
}

func main() {
	endpoint := "http://localhost:8080/"
	// Storage for url data
	data := url.Values{}
	fmt.Println("Please input long url")
	// read from console
	reader := bufio.NewReader(os.Stdin)
	long, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input line: %v\n", err)
		os.Exit(1)
	}
	long = strings.TrimSuffix(long, "\n")
	// fill container with data
	data.Set("url", long)
	// add HTTP-client
	client := NewHTTPClient()
	// write request
	request, err := makeRequest(endpoint, data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
	}
	// send request and receive response
	response, err := client.Do(request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		os.Exit(1)
	}
	// Print status code
	fmt.Println("Status code ", response.Status)
	defer response.Body.Close()
	// Read from response
	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(1)
	}
	// Print response
	fmt.Println(string(body))
}

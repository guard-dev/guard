package httpmiddleware

import (
	"fmt"
	"io"
	"net/http"
)

type HttpRequestStruct struct {
	Method  string
	Url     string
	Body    io.Reader
	Headers map[string]string
}

func HttpRequest(args HttpRequestStruct) ([]byte, error) {
	req, err := http.NewRequest(args.Method, args.Url, args.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to create request: " + err.Error())
	}

	for key, val := range args.Headers {
		req.Header.Set(key, val)
	}

	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch response: " + err.Error())
	}

	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)

	// Error out if response code is not 200 or 202.
	// But what if the response code is okay but not equal to 200 or 202?
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusAccepted && res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("Request failed: %d %s", res.StatusCode, responseBody)
	}

	if err != nil {
		return nil, fmt.Errorf("Failed to read response body: " + err.Error())
	}

	return responseBody, nil
}

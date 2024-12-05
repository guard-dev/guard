package utils

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io/ioutil"
	"math"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

// Used for retry logic.
func GetExponentialDelaySeconds(retryNumber int) int {
	delayTime := int(5 * math.Pow(2, float64(retryNumber)))
	return delayTime
}

// Wrapper for running commands that my not print out stderr normally otherwise
func ExecCommand(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	errMsg := stderr.String()

	if err != nil {
		return "", fmt.Errorf("Command failed: %s, %s", err.Error(), errMsg)
	}
	return string(output), nil
}

func ConvertUnixToTime(unixVal int64) time.Time {
	epochMillis := unixVal
	seconds := epochMillis / 1000
	nanoseconds := (epochMillis % 1000) * 1000000
	timeObj := time.Unix(seconds, nanoseconds)
	return timeObj
}

// function to uncompress a gzip blob
func UncompressGzip(blob []byte) ([]byte, error) {
	buf := bytes.NewBuffer(blob)
	r, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return ioutil.ReadAll(r)
}

func UnpackEventZlib(blob []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(blob))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return ioutil.ReadAll(reader)
}

// Helper function to check if the data is gzipped
func IsGzipped(data []byte) bool {
	return len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b
}

// Convert a 2D matrix to a 1D list of unique entries
func FlattenUniqueEntries(matrix [][]string) []string {
	uniqueMap := make(map[string]bool)
	var uniqueList []string

	for _, row := range matrix {
		for _, entry := range row {
			if _, found := uniqueMap[entry]; !found {
				uniqueMap[entry] = true
				uniqueList = append(uniqueList, entry)
			}
		}
	}

	return uniqueList
}

// Convert a list of urls to a list of unique endpoints.
func GetUniqueEndpoints(urls []string) []string {
	endpointSet := make(map[string]struct{})

	for _, rawURL := range urls {
		parsedURL, err := url.Parse(rawURL)
		if err != nil {
			fmt.Println("Error parsing URL:", err)
			continue
		}

		endpoint := strings.TrimRight(parsedURL.Path, "/")
		endpointSet[endpoint] = struct{}{}
	}

	uniqueEndpoints := make([]string, 0, len(endpointSet))
	for endpoint := range endpointSet {
		uniqueEndpoints = append(uniqueEndpoints, endpoint)
	}

	return uniqueEndpoints
}

// remove empty strings from a list
func RemoveEmptyEntries(input []string) []string {
	var result []string
	for _, str := range input {
		trimmed := strings.TrimSpace(str)
		if trimmed != "" && trimmed != "about:blank" && trimmed != "http://" && trimmed != "https://" {
			result = append(result, trimmed)
		}
	}
	return result
}

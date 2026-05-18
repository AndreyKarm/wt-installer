package wtlive

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
)

const (
	BaseURL     = "https://live.warthunder.com"
	RegularPath = "/api/feed/get_regular/"
	HeadPath    = "/api/feed/get_head/"
)

func createFormBody(fields map[string]string) (io.Reader, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, "", err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return body, writer.FormDataContentType(), nil
}

func GetFiltersFromAPI(fields map[string]string) (*ApiHeadResponse, error) {
	body, contentType, err := createFormBody(fields)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, BaseURL+HeadPath, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`const filters = (\{[\s\S]*?\});`)
	matches := re.FindSubmatch(bodyBytes)
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not find filters in response")
	}

	var result ApiHeadResponse
	if err := json.Unmarshal(matches[1], &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal filters JSON: %w", err)
	}

	return &result, nil
}

func GetFeed(fields map[string]string) (*ApiFeedResponse, error) {
	body, contentType, err := createFormBody(fields)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, BaseURL+RegularPath, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	var result ApiFeedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
)

// API Feed Types
type ApiFeedResponse struct {
	Status string `json:"status"`
	Data   Data   `json:"data"`
}

type Data struct {
	List []Post `json:"list"`
}

type Post struct {
	LangGroup   int     `json:"lang_group"`
	ID          int     `json:"id"`
	Type        string  `json:"type"`
	Author      Author  `json:"author"`
	Description string  `json:"description"`
	Images      []Image `json:"images"`
	File        File    `json:"file"`
}

type Author struct {
	ID       int    `json:"id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type Image struct {
	ID     int    `json:"id"`
	Type   string `json:"type"`
	Src    string `json:"src"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type File struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Link string `json:"link"`
	Type string `json:"type"`
	Size int    `json:"size"`
}

// API Head Structs
type ApiHeadResponse struct {
	VehicleCountry Filter `json:"vehicleCountry"`
	VehicleType    Filter `json:"vehicleType"`
	VehicleClass   Filter `json:"vehicleClass"`
	Vehicle        Filter `json:"vehicle"`
}

type Filter struct {
	Placeholder string    `json:"placeholder"`
	Variants    []Variant `json:"variants"`
}

type Variant struct {
	Separator bool                `json:"separator,omitempty"`
	Value     string              `json:"value,omitempty"`
	Name      string              `json:"name"`
	Count     int                 `json:"count,omitempty"`
	Dep       map[string][]string `json:"dep,omitempty"`
}

func CreateFormBody(fields map[string]string) (io.Reader, string, error) {
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
	url := baseUrl + head

	body, contentType, err := CreateFormBody(fields)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	client := &http.Client{}
	resp, err := client.Do(req)
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

	var filters ApiHeadResponse
	if err := json.Unmarshal(matches[1], &filters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return &filters, nil
}

func GetFeed(fields map[string]string) (*ApiFeedResponse, error) {
	url := baseUrl + regular

	body, contentType, err := CreateFormBody(fields)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	var apiResponse ApiFeedResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, err
	}

	return &apiResponse, nil
}

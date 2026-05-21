package main

import (
	"fmt"
	"time"

	g "github.com/AllenDang/giu"
)

var (
	FetchedPosts     []Post
	currentRequestID int64
	Filters          ApiHeadResponse

	IsLoading    bool
	LastLoadTime time.Time // Used to throttle requests

	Criteria = map[string]string{
		"content":        "camouflage",
		"sort":           "created",
		"user":           "",
		"searchString":   "",
		"page":           "0",
		"featured":       "0",
		"vehicleCountry": "",
		"vehicleType":    "",
		"vehicleClass":   "",
		"vehicle":        "",
	}

	Lang = "en"
)

func OnRequestHead() {
	snapshot := make(map[string]string, len(Criteria))
	for k, v := range Criteria {
		snapshot[k] = v
	}

	newFilters, err := GetFiltersFromAPI(Criteria)
	if err != nil {
		fmt.Println("Error fetching filters:", err)
		return
	}
	Filters = *newFilters
	fmt.Println("Filters reloaded!")
	g.Update()
}

func OpenSkin(id int) {
	url := fmt.Sprintf("%s/post/%d/%s/", BaseURL, id, Lang)
	fmt.Printf("Opening: %s\n", url)
	g.OpenURL(url)
}

package wtlive

import (
	"fmt"
	"sync/atomic"
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

func OnRequestData() {
	if IsLoading {
		return
	}
	IsLoading = true
	LastLoadTime = time.Now()

	defer func() {
		IsLoading = false
		g.Update()
	}()

	snapshot := make(map[string]string, len(Criteria))
	for k, v := range Criteria {
		snapshot[k] = v
	}

	myID := atomic.AddInt64(&currentRequestID, 1)

	FetchedPosts = []Post{}
	fmt.Printf("Fetching page %s...\n", snapshot["page"])
	g.Update()

	result, err := GetFeed(Criteria)
	if err != nil {
		fmt.Println("Error fetching feed:", err)
		return
	}

	if atomic.LoadInt64(&currentRequestID) != myID {
		fmt.Println("Discarding stale response.")
		return
	}

	if result == nil || len(result.Data.List) == 0 {
		fmt.Println("No posts found.")
		return
	}

	FetchedPosts = append(FetchedPosts, result.Data.List...)
}

func LoadNextPage() {
	if IsLoading {
		return
	}
	IsLoading = true
	LastLoadTime = time.Now()

	defer func() {
		IsLoading = false
		g.Update()
	}()

	snapshot := make(map[string]string, len(Criteria))
	for k, v := range Criteria {
		snapshot[k] = v
	}

	myID := atomic.AddInt64(&currentRequestID, 1)
	fmt.Printf("Loading more posts (page %s)...\n", snapshot["page"])

	result, err := GetFeed(Criteria)
	if err != nil {
		fmt.Println("Error fetching feed:", err)
		return
	}

	if atomic.LoadInt64(&currentRequestID) != myID {
		return
	}

	if result == nil || len(result.Data.List) == 0 {
		return
	}

	FetchedPosts = append(FetchedPosts, result.Data.List...)
}

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

func OnCamoClick(name string) {
	fmt.Printf("Selected camo: %s\n", name)
}

func OnInputRequest(input string) {
	fmt.Printf("Input: %s\n", input)
}

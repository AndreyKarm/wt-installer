package main

import (
	"fmt"
	"sync/atomic"

	g "github.com/AllenDang/giu"
)

func OnRequestData() {
	snapshot := make(map[string]string, len(criteria))
	for k, v := range criteria {
		snapshot[k] = v
	}

	myID := atomic.AddInt64(&currentRequestID, 1)

	fetchedPosts = []Post{}
	fmt.Printf("Fetching page %s...\n", snapshot["page"])
	g.Update()

	result, err := GetFeed(criteria)
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

	fetchedPosts = append(fetchedPosts, result.Data.List...)
	fmt.Printf("Done! Fetched %d posts.\n", len(fetchedPosts))
	g.Update()
}

func OnRequestHead() {
	snapshot := make(map[string]string, len(criteria))
	for k, v := range criteria {
		snapshot[k] = v
	}

	newFilters, err := GetFiltersFromAPI(criteria)
	if err != nil {
		fmt.Println("Error fetching filters:", err)
		return
	}
	filters = *newFilters
	fmt.Println("Filters reloaded!")
	g.Update()
}

func OpenSkin(id int) {
	url := fmt.Sprintf("%s/post/%d/%s/", baseURL, id, lang)
	fmt.Printf("Opening: %s\n", url)
	g.OpenURL(url)
}

func OnCamoClick(name string) {
	fmt.Printf("Selected camo: %s\n", name)
}

func OnInputRequest(input string) {
	fmt.Printf("Input: %s\n", input)
}

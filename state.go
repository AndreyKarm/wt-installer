package main

import (
	"fmt"
	"sync/atomic"
	"time"

	g "github.com/AllenDang/giu"
)

var (
	wnd *g.MasterWindow

	CurrentConfig  *Config
	DownloadStatus = map[int]string{}
)

func OnRequestData() {
	fmt.Println(Criteria)

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

package main

import (
	"image"

	g "github.com/AllenDang/giu"
)

const (
	baseURL     = "https://live.warthunder.com"
	regularPath = "/api/feed/get_regular/"
	headPath    = "/api/feed/get_head/"
	lang        = "en"
)

var (
	// UI state
	activeTab       int32 = 0
	countrySelected int32
	typeSelected    int32
	classSelected   int32
	vehicleSelected int32
	feedSort        int32
	searchInput     string

	// Texture
	rgba        *image.RGBA
	tex         *g.Texture
	fallbackTex *g.Texture

	// Config
	currentConfig *Config
	skinPathInput string

	// Filter data
	filters ApiHeadResponse

	// Feed state
	fetchedPosts     []Post
	downloadStatus   = map[int]string{}
	skinToDelete     string
	currentPage      int32 = 0
	currentRequestID int64

	criteria = map[string]string{
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
)

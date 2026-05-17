package main

// Feed types

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
	Created     int64   `json:"created"`
	Author      Author  `json:"author"`
	Description string  `json:"description"`
	Downloads   int     `json:"downloads"`
	Likes       int     `json:"likes"`
	Views       int     `json:"views"`
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

// Filter types

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

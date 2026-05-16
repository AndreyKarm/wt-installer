package main

import (
	"fmt"
	"os"
	"time"

	g "github.com/AllenDang/giu"
)

func OnRequestData() {
	go func() {
		fmt.Println("Fetching all posts...")
		fetchedPosts = []Post{}
		g.Update()

		result, err := GetFeed(criteria)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		if result == nil || len(result.Data.List) == 0 {
			fmt.Println("No posts found.")
			return
		}

		fetchedPosts = append(fetchedPosts, result.Data.List...)
		g.Update()
		fmt.Printf("Done! Fetched %d total posts.\n", len(fetchedPosts))
	}()
}

func OnRequestHead() {
	go func() {
		newFilters, err := GetFiltersFromAPI(criteria)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		filters = *newFilters
		g.Update()
		fmt.Println("Filters loaded!")
	}()
}

func OpenSkin(id int) {
	url := fmt.Sprintf("%s/post/%v/%s/", baseUrl, id, lang)
	fmt.Printf("Opening browser: %s\n", url)
	g.OpenURL(url)
}

func OnCamoClick(name string) {
	fmt.Printf("Camo: %s\n", name)
}

func BuildCamoList() []g.Widget {
	var widgets []g.Widget

	for i := range fetchedPosts {
		post := fetchedPosts[i]

		if len(post.Images) == 0 {
			continue
		}

		status := downloadStatus[post.ID]

		var statusWidget g.Widget
		if status != "" {
			statusWidget = g.Label(status)
		} else {
			statusWidget = g.Label("")
		}

		widget := g.Row(
			g.ImageWithURL(post.Images[0].Src).
				Timeout(5*time.Second).
				Size(100, 100).
				OnClick(func() {
					OpenSkin(post.LangGroup)
				}).
				LayoutForLoading(
					g.Child().Size(100, 100).Layout(g.Layout{
						g.Label("Loading..."),
					}),
				).
				LayoutForFailure(
					g.ImageWithFile("./media/fallback.png").Size(100, 100),
				),
			g.Column(
				g.Selectable(
					fmt.Sprintf(
						"%v: %s - %s",
						post.ID, post.Author.Nickname, post.File.Name,
					),
				).OnClick(func() {
					OpenSkin(post.LangGroup)
				}),
				g.Row(
					g.Button(fmt.Sprintf("Download##dl%d", post.ID)).
						OnClick(func() {
							DownloadSkin(post)
						}),
					statusWidget,
				),
			),
		)

		widgets = append(widgets, widget)
	}

	return widgets
}

func LoadedCamoList() []g.Widget {
	var widgets []g.Widget

	config, err := LoadConfig()
	if err != nil {
		widgets = append(widgets, g.Label(fmt.Sprintf("Config error: %v", err)))
		return widgets
	}

	entries, err := os.ReadDir(config.UserSkins)
	if err != nil {
		widgets = append(
			widgets,
			g.Label(fmt.Sprintf("Folder read error: %v", err)),
		)
		return widgets
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		widgets = append(widgets, g.Row(
			g.Selectable(name).OnClick(func() {
				OnCamoClick(name)
			}),
			g.Button(fmt.Sprintf("Delete##del%s", name)).OnClick(func() {
				skinToDelete = name
				g.OpenPopup(deletePopupID)
			}),
		))
	}

	return widgets
}

func OnInputRequest(input string) {
	fmt.Printf("World: %s\n", input)
}

func LogChange() {
	fmt.Println(textInput)
}

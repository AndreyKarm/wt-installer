package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	g "github.com/AllenDang/giu"
)

var (
	selectedImagePost  *Post
	selectedImageIndex int
	showImageModal     bool
)

func BuildCamoList() []g.Widget {
	var widgets []g.Widget
	const ImageSizeMultiplier = 0.5

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

		colItems := []g.Widget{
			g.Column(
				g.Selectable(fmt.Sprintf(
					"Author: %s",
					post.Author.Nickname,
				)).OnClick(func() {
					g.OpenURL(fmt.Sprintf(
						"https://live.warthunder.com/user/%d",
						post.Author.ID,
					))
				}),

				g.Label(fmt.Sprintf("Date Created: %s", time.Unix(post.Created, 0).Format("2006-01-02 15:04:05"))),

				g.Label(fmt.Sprintf(
					"Downloads: %v. Likes: %v. Views: %v",
					post.Downloads,
					post.Likes,
					post.Views,
				)),
			),

			g.Row(
				g.Button(fmt.Sprintf("Download##dl%d", post.ID)).
					Size(100, 0).
					OnClick(func() { DownloadSkin(post) }),
				statusWidget,
			),
		}

		// TODO: Change the hashtag list to something better, right now it calculates the length of each tag for the wrapping because gui doesn't support wrapping on buttons
		if hashtags := ExtractHashtags(post.Description); len(hashtags) > 0 {
			var tagRows []g.Widget
			var currentRow []g.Widget

			currentRow = append(currentRow, g.Label("Tags:"))
			rowLen := 5 // Initial weight for "Tags:" label

			for j, tag := range hashtags {
				t := tag
				btn := g.Button(fmt.Sprintf("%s##tag%d_%d", t, post.ID, j)).
					OnClick(func() {
						searchInput = strings.TrimPrefix(t, "#")
						criteria["searchString"] = t
						currentPage = 0
						criteria["page"] = "0"
						go OnRequestData()
					})

				itemLen := len(t) + 4

				if rowLen+itemLen > 70 {
					tagRows = append(tagRows, g.Row(currentRow...))
					currentRow = []g.Widget{btn}
					rowLen = itemLen
				} else {
					currentRow = append(currentRow, btn)
					rowLen += itemLen
				}
			}
			if len(currentRow) > 0 {
				tagRows = append(tagRows, g.Row(currentRow...))
			}

			colItems = append(colItems, g.Separator(), g.Column(tagRows...))
		}

		widgets = append(widgets, g.Row(
			g.ImageWithURL(post.Images[0].Src).
				Timeout(5*time.Second).
				Size(
					float32(post.Images[0].Width)*ImageSizeMultiplier,
					float32(post.Images[0].Height)*ImageSizeMultiplier,
				).
				OnClick(func() {
					g.OpenPopup(fmt.Sprintf("Images %d", post.ID))
				}).
				LayoutForLoading(
					g.Image(fallbackTex).Size(100, 100),
				).
				LayoutForFailure(
					g.Image(fallbackTex).Size(100, 100),
				),
			g.PopupModal(fmt.Sprintf("Images %d", post.ID)).Layout(
				g.Custom(func() {
					if g.IsKeyPressed(g.KeyEscape) {
						g.CloseCurrentPopup()
					}
				}),
				g.Column(GetImagesFromPost(&post)...),
				g.Button("Close##closeimg").OnClick(func() {
					g.CloseCurrentPopup()
				}),
			),
			g.Column(colItems...),
		))
		widgets = append(widgets, g.Separator())
	}

	return widgets
}

func LoadedCamoList() []g.Widget {
	var widgets []g.Widget

	config, err := LoadConfig()
	if err != nil {
		return append(widgets, g.Label(fmt.Sprintf("Config error: %v", err)))
	}

	entries, err := os.ReadDir(config.UserSkins)
	if err != nil {
		return append(
			widgets,
			g.Label(fmt.Sprintf("Folder read error: %v", err)),
		)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		directory := filepath.Join(config.UserSkins, name)
		if skinToDelete == name {
			widgets = append(widgets, g.Row(
				g.Label(name),
				g.Button("Open folder").OnClick(func() {
					exec.Command("explorer", directory).Start()
				}),
				g.Label("  Delete this skin?"),
				g.Button(fmt.Sprintf("Yes, Delete##yes%s", name)).
					Size(100, 0).
					OnClick(func() {
						skinToDelete = ""
						DeleteSkin(name)
					}),
				g.Button(fmt.Sprintf("Cancel##no%s", name)).
					Size(80, 0).
					OnClick(func() {
						skinToDelete = ""
					}),
			))
		} else {
			widgets = append(widgets, g.Row(
				g.Label(name),
				g.Button("Open folder").OnClick(func() {
					exec.Command("explorer", directory).Start()
				}),
				g.Button(fmt.Sprintf("Delete##del%s", name)).
					Size(80, 0).
					OnClick(func() {
						skinToDelete = name
					}),
			))
		}
	}

	return widgets
}

func GetImagesFromPost(post *Post) []g.Widget {
	var widgets []g.Widget
	const ImageSizeMultiplier = 1

	for i, img := range post.Images {
		widgets = append(widgets, g.ImageWithURL(img.Src).
			Size(
				float32(img.Width)*ImageSizeMultiplier,
				float32(img.Height)*ImageSizeMultiplier,
			).
			OnClick(func() {
				selectedImagePost = post
				selectedImageIndex = i
				showImageModal = true
			}),
		)
	}

	return widgets
}

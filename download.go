package main

import (
	"fmt"
	"liotom/installer/installer"
	"liotom/installer/wtlive"
	"strings"
	"time"

	g "github.com/AllenDang/giu"
)

var (
	currentPage     int32 = 0
	searchInput     string
	feedSort        int32
	countrySelected int32
	typeSelected    int32
	classSelected   int32
	vehicleSelected int32

	selectedImagePost  *wtlive.Post
	selectedImageIndex int
	showImageModal     bool
)

func DownloadPage() []g.Widget {
	filteredCountries := wtlive.FilterVariants(wtlive.Filters.VehicleCountry.Variants, wtlive.Criteria)
	filteredTypes := wtlive.FilterVariants(wtlive.Filters.VehicleType.Variants, wtlive.Criteria)
	filteredClasses := wtlive.FilterVariants(wtlive.Filters.VehicleClass.Variants, wtlive.Criteria)
	filteredVehicles := wtlive.FilterVariants(wtlive.Filters.Vehicle.Variants, wtlive.Criteria)

	sortOptions := []string{
		"Recent", "Popular", "Most commented", "Most downloaded",
	}
	sortValues := map[string]string{
		"Recent":          "created",
		"Popular":         "rating",
		"Most commented":  "comments",
		"Most downloaded": "downloads",
	}

	return []g.Widget{
		g.Row(
			g.Label("Search"),
			g.InputText(&searchInput).
				Hint("e.g. historical ussr").
				Size(240),
			g.Custom(func() {
				if g.IsKeyPressed(g.KeyEnter) && searchInput != "" {
					onSearch()
				}
			}),
			g.Label(wtlive.WordsToHashtags(searchInput)),
			g.Button("Search##searchbtn").OnClick(func() {
				if searchInput == "" {
					return
				}
				onSearch()
			}),
			g.Button("Clear##clearbtn").OnClick(func() {
				if searchInput == "" {
					return
				}
				searchInput = ""
				wtlive.Criteria["searchString"] = ""
				currentPage = 0
				wtlive.Criteria["page"] = "0"
				go wtlive.OnRequestData()
			}),
		),

		g.Row(
			g.Label("Sort"),
			g.Combo("", sortOptions[feedSort], sortOptions, &feedSort).
				OnChange(func() {
					wtlive.Criteria["sort"] = sortValues[sortOptions[feedSort]]
					go wtlive.OnRequestData()
				}).Size(160),

			g.Label("Country"),
			g.Combo(
				"", wtlive.GetLabel(filteredCountries, countrySelected),
				wtlive.GetItems(filteredCountries), &countrySelected,
			).OnChange(func() {
				if int(countrySelected) < len(filteredCountries) {
					wtlive.Criteria["vehicleCountry"] =
						filteredCountries[countrySelected].Value
					go wtlive.OnRequestData()
				}
			}).Size(160),

			g.Label("Type"),
			g.Combo(
				"", wtlive.GetLabel(filteredTypes, typeSelected),
				wtlive.GetItems(filteredTypes), &typeSelected,
			).OnChange(func() {
				if int(typeSelected) < len(filteredTypes) {
					wtlive.Criteria["vehicleType"] = filteredTypes[typeSelected].Value
					classSelected = 0
					wtlive.Criteria["vehicleClass"] = ""
					vehicleSelected = 0
					wtlive.Criteria["vehicle"] = ""
					go wtlive.OnRequestData()
				}
			}).Size(160),

			g.Label("Class"),
			g.Combo(
				"", wtlive.GetLabel(filteredClasses, classSelected),
				wtlive.GetItems(filteredClasses), &classSelected,
			).OnChange(func() {
				if int(classSelected) < len(filteredClasses) {
					wtlive.Criteria["vehicleClass"] =
						filteredClasses[classSelected].Value
					vehicleSelected = 0
					wtlive.Criteria["vehicle"] = ""
					go wtlive.OnRequestData()
				}
			}).Size(160),

			g.Label("Vehicle"),
			g.Combo(
				"", wtlive.GetLabel(filteredVehicles, vehicleSelected),
				wtlive.GetItems(filteredVehicles), &vehicleSelected,
			).OnChange(func() {
				if int(vehicleSelected) < len(filteredVehicles) {
					wtlive.Criteria["vehicle"] =
						filteredVehicles[vehicleSelected].Value
					go wtlive.OnRequestData()
				}
			}).Size(160).Filter(true),
		),

		g.Separator(),
		g.Row(
			g.Button("< Prev").OnClick(func() {
				if currentPage > 0 {
					currentPage--
					wtlive.Criteria["page"] = fmt.Sprintf("%d", currentPage)
					go wtlive.OnRequestData()
				}
			}),
			g.Label(fmt.Sprintf("Page %d", currentPage+1)),
			g.Button("Next >").OnClick(func() {
				currentPage++
				wtlive.Criteria["page"] = fmt.Sprintf("%d", currentPage)
				go wtlive.OnRequestData()
			}),
		),
		g.Child().Layout(
			g.Column(DownloadSkin()...),
		),
	}
}

func DownloadSkin() []g.Widget {
	var widgets []g.Widget
	const ImageSizeMultiplier = 0.5

	for i := range wtlive.FetchedPosts {
		post := wtlive.FetchedPosts[i]
		if len(post.Images) == 0 {
			continue
		}

		status := installer.DownloadStatus[post.ID]
		var statusString string
		if status != "" {
			statusString = status
		} else {
			statusString = "Download"
		}

		colItems := []g.Widget{
			g.Column(
				g.Row(
					g.Label("Author:"),
					g.Button(post.Author.Nickname).
						OnClick(func() {
							g.OpenURL(fmt.Sprintf(
								"https://live.warthunder.com/user/%d",
								post.Author.ID,
							))
						}),
				),

				g.Label(fmt.Sprintf("Date Created: %s", time.Unix(post.Created, 0).Format("01/02/2006 15:04:05"))),

				g.Label(fmt.Sprintf(
					"Downloads: %v. Likes: %v. Views: %v",
					post.Downloads,
					post.Likes,
					post.Views,
				)),
			),

			g.Row(
				g.Button(fmt.Sprintf("%s##dl%d", statusString, post.ID)).
					Size(100, 0).
					OnClick(func() { installer.DownloadSkin(post) }),
				g.Align(g.AlignRight).To(
					g.Button("Link").OnClick(func() {
						g.OpenURL(fmt.Sprintf(
							"%d/post/%d/%d/",
							wtlive.BaseURL,
							post.LangGroup,
							wtlive.Lang,
						))
					}).Size(100, 0),
				),
			),
		}

		if hashtags := wtlive.ExtractHashtags(post.Description); len(hashtags) > 0 {
			var width, _ = wnd.GetSize()

			imgWidth := float32(post.Images[0].Width) * ImageSizeMultiplier

			const safetyMargin float32 = 40.0
			availableWidth := float32(width) - imgWidth - safetyMargin

			var tagRows []g.Widget
			var currentRow []g.Widget
			var currentX float32 = 0.0

			for j, tag := range hashtags {
				t := tag

				textWidth, _ := g.CalcTextSize(t)
				btnWidth := textWidth + 16.0

				if currentX+btnWidth > availableWidth && len(currentRow) > 0 {
					tagRows = append(tagRows, g.Row(currentRow...))
					currentRow = nil
					currentX = 0.0
				}

				btn := g.Button(fmt.Sprintf("%s##tag%d_%d", t, post.ID, j)).
					OnClick(func() {
						searchInput = strings.TrimPrefix(t, "#")
						wtlive.Criteria["searchString"] = t
						currentPage = 0
						wtlive.Criteria["page"] = "0"
						go wtlive.OnRequestData()
					})

				currentRow = append(currentRow, btn)
				currentX += btnWidth + 8.0
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
					g.Image(FallbackTex).Size(100, 100),
				).
				LayoutForFailure(
					g.Image(FallbackTex).Size(100, 100),
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

func GetImagesFromPost(post *wtlive.Post) []g.Widget {
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

func onSearch() {
	wtlive.Criteria["searchString"] = wtlive.WordsToHashtags(searchInput)
	currentPage = 0
	wtlive.Criteria["page"] = "0"
	go wtlive.OnRequestData()
}

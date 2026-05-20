package main

import (
	"fmt"
	"liotom/installer/installer"
	"liotom/installer/wtlive"
	"strings"
	"time"

	imgui "github.com/AllenDang/cimgui-go/imgui"
	g "github.com/AllenDang/giu"
)

var (
	currentPage     int32 = 0
	hashtagInput    string
	feedSort        int32
	countrySelected int32
	typeSelected    int32
	classSelected   int32
	vehicleSelected int32

	filterUniversal bool

	selectedImagePost  *wtlive.Post
	selectedImageIndex int
	showImageModal     bool

	pixelsToEnd           float32
	pixelsToLoadThreshold int = 500
	scrollToTop           bool
)

const (
	ImageSizeMultiplier     = 0.5
	ViewImageSizeMultiplier = 1.5
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
			g.InputText(&hashtagInput).
				Hint("e.g. historical ussr").
				Size(400),
			g.Custom(func() {
				if g.IsKeyPressed(g.KeyEnter) && hashtagInput != "" {
					onSearch()
				}
			}),
			// g.Label(wtlive.WordsToHashtags(searchInput)),
			g.Button("Search##searchbtn").OnClick(func() {
				if hashtagInput == "" {
					return
				}
				onSearch()
			}),
			g.Button("Clear##clearbtn").OnClick(func() {
				if hashtagInput == "" {
					return
				}
				hashtagInput = ""
				wtlive.Criteria["searchString"] = ""
				currentPage = 0
				wtlive.Criteria["page"] = "0"

				scrollToTop = true
				go wtlive.OnRequestData()
			}),

			g.Checkbox("Filter Universal", &filterUniversal).OnChange(func() {
				wtlive.Criteria["page"] = "0"
				currentPage = 0

				scrollToTop = true
				go wtlive.OnRequestData()
			}),
		),

		g.Row(
			g.Label("Sort"),
			g.Combo("", sortOptions[feedSort], sortOptions, &feedSort).
				OnChange(func() {
					wtlive.Criteria["sort"] = sortValues[sortOptions[feedSort]]

					wtlive.Criteria["page"] = "0"
					currentPage = 0

					scrollToTop = true
					go wtlive.OnRequestData()
				}).Size(160),

			g.Label("Country"),
			g.Combo(
				"", wtlive.GetLabel(filteredCountries, countrySelected),
				wtlive.GetItems(filteredCountries), &countrySelected,
			).OnChange(func() {
				if int(countrySelected) < len(filteredCountries) {
					wtlive.Criteria["vehicleCountry"] = filteredCountries[countrySelected].Value

					wtlive.Criteria["page"] = "0"
					currentPage = 0

					scrollToTop = true
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

					wtlive.Criteria["page"] = "0"
					currentPage = 0

					scrollToTop = true
					go wtlive.OnRequestData()
				}
			}).Size(160),

			g.Label("Class"),
			g.Combo(
				"", wtlive.GetLabel(filteredClasses, classSelected),
				wtlive.GetItems(filteredClasses), &classSelected,
			).OnChange(func() {
				if int(classSelected) < len(filteredClasses) {
					wtlive.Criteria["vehicleClass"] = filteredClasses[classSelected].Value
					wtlive.Criteria["vehicle"] = ""
					vehicleSelected = 0

					wtlive.Criteria["page"] = "0"
					currentPage = 0

					scrollToTop = true
					go wtlive.OnRequestData()
				}
			}).Size(160),

			g.Label("Vehicle"),
			g.Combo(
				"", wtlive.GetLabel(filteredVehicles, vehicleSelected),
				wtlive.GetItems(filteredVehicles), &vehicleSelected,
			).OnChange(func() {
				if int(vehicleSelected) < len(filteredVehicles) {
					wtlive.Criteria["vehicle"] = filteredVehicles[vehicleSelected].Value

					wtlive.Criteria["page"] = "0"
					currentPage = 0

					scrollToTop = true
					go wtlive.OnRequestData()
				}
			}).Size(160).Filter(true),
		),

		g.Separator(),
		g.Child().Layout(
			g.Custom(func() {
				if scrollToTop {
					imgui.SetScrollYFloat(0)
					scrollToTop = false
				}
				y := imgui.ScrollY()
				maxY := imgui.ScrollMaxY()
				pixelsToEnd = maxY - y

				if maxY > 0 && pixelsToEnd < 500 && !wtlive.IsLoading && time.Since(wtlive.LastLoadTime) > 2*time.Second {
					currentPage++
					wtlive.Criteria["page"] = fmt.Sprintf("%d", currentPage)
					go wtlive.LoadNextPage()
				}
			}),
			g.Column(PostWidget()...),
			g.Custom(func() {
				if wtlive.IsLoading && len(wtlive.FetchedPosts) > 0 {
					g.Label("Loading more posts...").Build()
				}
			}),
		),
	}
}

func PostWidget() []g.Widget {
	var widgets []g.Widget

	for i := range wtlive.FetchedPosts {
		post := wtlive.FetchedPosts[i]
		if len(post.Images) == 0 {
			continue
		}

		isUniversal := strings.Contains(strings.ToLower(post.Description), "#universal")
		if filterUniversal && isUniversal {
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
								"%s/user/%d",
								wtlive.BaseURL,
								post.Author.ID,
							))
						}),
					g.Align(g.AlignRight).To(
						g.Row(
							g.Button(fmt.Sprintf("%s##dl%d", statusString, post.ID)).
								Size(150, 0).
								OnClick(func() { installer.DownloadSkin(post) }),
							g.Button(fmt.Sprintf("Link##link%d", post.ID)).
								Size(100, 0).
								OnClick(func() {
									g.OpenURL(fmt.Sprintf(
										"%s/post/%v/%s/\n",
										wtlive.BaseURL,
										post.LangGroup,
										wtlive.Lang,
									))
								}),
						),
					),
				),

				g.Label(fmt.Sprintf("Date Created: %s", time.Unix(post.Created, 0).Format("02/01/2005 15:04:05"))),

				g.Label(fmt.Sprintf(
					"Downloads: %v. Likes: %v. Views: %v",
					post.Downloads,
					post.Likes,
					post.Views,
				)),
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
						hashtagInput = strings.TrimPrefix(t, "#")
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

		var imgWidth, imgHeight = post.Images[0].Width, post.Images[0].Height
		const diff float32 = (384.0 / 600.0) * ImageSizeMultiplier
		const fallBackWidth, fallBackHeight float32 = 600.0 * diff, 400.0 * diff

		widgets = append(widgets, g.Row(
			g.ImageWithURL(post.Images[0].Src).
				Timeout(5*time.Second).
				Size(
					float32(imgWidth)*ImageSizeMultiplier,
					float32(imgHeight)*ImageSizeMultiplier,
				).
				OnClick(func() {
					g.OpenPopup(fmt.Sprintf("Images##img%d", post.ID))
				}).
				LayoutForLoading(
					g.Image(FallbackTex).Size(fallBackWidth, fallBackHeight),
				).
				LayoutForFailure(
					g.Image(FallbackTex).Size(fallBackWidth, fallBackHeight),
				),
			g.PopupModal(fmt.Sprintf("Images##img%d", post.ID)).Layout(
				g.Custom(func() {
					if g.IsKeyPressed(g.KeyEscape) {
						g.CloseCurrentPopup()
					}
				}),
				g.Child().
					Size(
						(float32(imgWidth)*ViewImageSizeMultiplier)+16,
						(float32(imgHeight)*ViewImageSizeMultiplier)+16,
					).
					Layout(
						g.Column(GetImagesFromPost(&post)...),
					),
				// g.Column(GetImagesFromPost(&post)...),
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

	for i, img := range post.Images {
		widgets = append(widgets, g.ImageWithURL(img.Src).
			Size(
				float32(img.Width)*ViewImageSizeMultiplier,
				float32(img.Height)*ViewImageSizeMultiplier,
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
	wtlive.Criteria["searchString"] = wtlive.WordsToHashtags(hashtagInput)
	wtlive.Criteria["page"] = "0"
	currentPage = 0
	scrollToTop = true
	go wtlive.OnRequestData()
}

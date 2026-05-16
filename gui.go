package main

import (
	"fmt"
	"image"

	g "github.com/AllenDang/giu"
)

var (
	activeTab int32 = 0

	filters         ApiHeadResponse
	countrySelected int32
	typeSelected    int32
	classSelected   int32
	vehicleSelected int32
	feedSort        int32
	textInput       string

	// Config
	currentConfig *Config
	skinPathInput string

	rgba *image.RGBA
	tex  *g.Texture

	// State
	fetchedPosts   []Post
	downloadStatus = map[int]string{}
	skinToDelete   string
	currentPage    int32 = 0

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

	// Warthunder Live API
	baseUrl = "https://live.warthunder.com"
	regular = "/api/feed/get_regular/"
	head    = "/api/feed/get_head/"
	lang    = "en"
)

const deletePopupID = "Confirm Delete##deleteModal"

func FilterVariants(variants []Variant, criteria map[string]string) []Variant {
	var filtered []Variant
	for _, v := range variants {
		if v.Separator {
			continue
		}
		if v.Value == "any" {
			filtered = append(filtered, v)
			continue
		}
		if len(v.Dep) == 0 {
			filtered = append(filtered, v)
			continue
		}
		match := true
		for depKey, depValues := range v.Dep {
			selectedVal := criteria[depKey]
			if selectedVal == "" {
				continue
			}
			valMatch := false
			for _, dv := range depValues {
				if dv == selectedVal {
					valMatch = true
					break
				}
			}
			if !valMatch {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func GetLabel(variants []Variant, idx int32) string {
	if len(variants) == 0 {
		return "Loading..."
	}
	if idx < 0 || int(idx) >= len(variants) {
		return variants[0].Name
	}
	return variants[idx].Name
}

func GetItems(variants []Variant) []string {
	items := make([]string, len(variants))
	for i, v := range variants {
		if v.Count > 0 {
			items[i] = fmt.Sprintf("%s (%v)", v.Name, v.Count)
		} else {
			items[i] = v.Name
		}
	}
	return items
}

func loop() {
	filteredCountries := FilterVariants(filters.VehicleCountry.Variants, criteria)
	filteredTypes := FilterVariants(filters.VehicleType.Variants, criteria)
	filteredClasses := FilterVariants(filters.VehicleClass.Variants, criteria)
	filteredVehicles := FilterVariants(filters.Vehicle.Variants, criteria)

	sortOptions := []string{
		"Recent", "Popular", "Most commented", "Most downloaded",
	}
	sortValues := map[string]string{
		"Recent":          "created",
		"Popular":         "rating",
		"Most commented":  "comments",
		"Most downloaded": "downloads",
	}

	g.SingleWindow().Layout(
		g.Label("WarThunder Camo Browser"),

		g.TabBar().TabItems(
			g.TabItem("Download").Layout(
				g.Row(
					g.Label("Sort"),
					g.Combo("", sortOptions[feedSort], sortOptions, &feedSort).
						OnChange(func() {
							criteria["sort"] = sortValues[sortOptions[feedSort]]
							go OnRequestData()
						}).Size(160),

					g.Label("Country"),
					g.Combo(
						"", GetLabel(filteredCountries, countrySelected),
						GetItems(filteredCountries), &countrySelected,
					).OnChange(func() {
						if int(countrySelected) < len(filteredCountries) {
							criteria["vehicleCountry"] =
								filteredCountries[countrySelected].Value
							go OnRequestData()
						}
					}).Size(160),

					g.Label("Type"),
					g.Combo(
						"", GetLabel(filteredTypes, typeSelected),
						GetItems(filteredTypes), &typeSelected,
					).OnChange(func() {
						if int(typeSelected) < len(filteredTypes) {
							criteria["vehicleType"] = filteredTypes[typeSelected].Value
							classSelected = 0
							criteria["vehicleClass"] = ""
							vehicleSelected = 0
							criteria["vehicle"] = ""
							go OnRequestData()
						}
					}).Size(160),

					g.Label("Class"),
					g.Combo(
						"", GetLabel(filteredClasses, classSelected),
						GetItems(filteredClasses), &classSelected,
					).OnChange(func() {
						if int(classSelected) < len(filteredClasses) {
							criteria["vehicleClass"] =
								filteredClasses[classSelected].Value
							vehicleSelected = 0
							criteria["vehicle"] = ""
							go OnRequestData()
						}
					}).Size(160),

					g.Label("Vehicle"),
					g.Combo(
						"", GetLabel(filteredVehicles, vehicleSelected),
						GetItems(filteredVehicles), &vehicleSelected,
					).OnChange(func() {
						if int(vehicleSelected) < len(filteredVehicles) {
							criteria["vehicle"] = filteredVehicles[vehicleSelected].Value
							go OnRequestData()
						}
					}).Size(160).Filter(true),
				),

				g.Separator(),
				g.Row(
					g.Button("< Prev").OnClick(func() {
						if currentPage > 0 {
							currentPage--
							criteria["page"] = fmt.Sprintf("%d", currentPage)
							go OnRequestData()
						}
					}),
					g.Label(fmt.Sprintf("Page %d", currentPage+1)),
					g.Button("Next >").OnClick(func() {
						currentPage++
						criteria["page"] = fmt.Sprintf("%d", currentPage)
						go OnRequestData()
					}),
				),
				g.Child().Layout(
					g.Column(BuildCamoList()...),
				),
			),

			g.TabItem("Manage Skins").Layout(
				g.Child().Layout(
					g.Column(LoadedCamoList()...),
				),
			),

			g.TabItem("Settings").Layout(
				g.Label("Configuration"),
				g.InputText(&skinPathInput),
				g.Button("Save Settings").OnClick(func() {
					currentConfig.UserSkins = skinPathInput
					if err := SaveConfig(currentConfig); err != nil {
						fmt.Println("Error saving config:", err)
					} else {
						fmt.Println("Settings saved!")
					}
				}),
			),
		),

		// Delete confirmation modal
		g.PopupModal(deletePopupID).Layout(
			g.Label(fmt.Sprintf("Delete '%s'?", skinToDelete)),
			g.Label("This will permanently remove the skin folder."),
			g.Dummy(0, 8),
			g.Row(
				g.Button("Delete").OnClick(func() {
					name := skinToDelete
					skinToDelete = ""
					DeleteSkin(name)
					g.CloseCurrentPopup()
				}),
				g.Button("Cancel").OnClick(func() {
					skinToDelete = ""
					g.CloseCurrentPopup()
				}),
			),
		),
	)
}

func main() {
	rgba, err := g.LoadImage("./res/app_win.jpg")
	if err != nil {
		fmt.Println("Error loading fallback image:", err)
	}

	wnd := g.NewMasterWindow(
		"WTLive Installer", 1200, 900, g.MasterWindowFlagsTransparent,
	)

	if rgba != nil {
		g.EnqueueNewTextureFromRgba(rgba, func(t *g.Texture) {
			tex = t
		})
	}

	cfg, err := LoadConfig()
	if err != nil {
		fmt.Println("Error loading config, using defaults:", err)
		cfg = GetDefaultConfig()
	}
	currentConfig = cfg
	skinPathInput = currentConfig.UserSkins

	go func() {
		data, err := GetFiltersFromAPI(criteria)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		filters = *data
		fmt.Println("Filters loaded!")
		g.Update()
	}()

	go OnRequestData()

	wnd.Run(loop)
}

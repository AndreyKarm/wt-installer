package main

import (
	"fmt"

	g "github.com/AllenDang/giu"
)

func onSearch() {
	criteria["searchString"] = WordsToHashtags(searchInput)
	currentPage = 0
	criteria["page"] = "0"
	go OnRequestData()
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
		g.TabBar().TabItems(
			g.TabItem("Download").Layout(
				g.Row(
					g.Label("Search"),
					g.InputText(&searchInput).
						Hint("e.g. historical ussr").
						Size(240),
					g.Custom(func() {
						if g.IsKeyPressed(g.KeyEnter) {
							onSearch()
						}
					}),
					g.Label(WordsToHashtags(searchInput)),
					g.Button("Search##searchbtn").OnClick(onSearch),
					g.Button("Clear##clearbtn").OnClick(func() {
						searchInput = ""
						criteria["searchString"] = ""
						currentPage = 0
						criteria["page"] = "0"
						go OnRequestData()
					}),
				),

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
							criteria["vehicle"] =
								filteredVehicles[vehicleSelected].Value
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
	)
}

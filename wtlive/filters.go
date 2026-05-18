package wtlive

import "fmt"

func FilterVariants(variants []Variant, criteria map[string]string) []Variant {
	var filtered []Variant
	for _, v := range variants {
		if v.Separator {
			continue
		}
		if v.Value == "any" || len(v.Dep) == 0 {
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
			items[i] = fmt.Sprintf("%s (%d)", v.Name, v.Count)
		} else {
			items[i] = v.Name
		}
	}
	return items
}

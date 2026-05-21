package main

import (
	"fmt"
	"regexp"
	"strings"
)

var hashtagRe = regexp.MustCompile(`class="WTL-Embed-Hashtag">(#\w+)<\/a>`)

func ExtractHashtags(description string) []string {
	matches := hashtagRe.FindAllStringSubmatch(description, -1)
	seen := make(map[string]bool, len(matches))
	result := make([]string, 0, len(matches))
	for _, m := range matches {
		tag := strings.ToLower(m[1])
		if !seen[tag] {
			seen[tag] = true
			result = append(result, tag)
		}
	}
	return result
}

func WordsToHashtags(input string) string {
	if strings.TrimSpace(input) == "" {
		return ""
	}
	words := strings.Fields(input)
	tags := make([]string, 0, len(words))
	for _, w := range words {
		if strings.HasPrefix(w, "#") {
			tags = append(tags, w)
		} else {
			tags = append(tags, "#"+w)
		}
	}
	return strings.Join(tags, " ")
}

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

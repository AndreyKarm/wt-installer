package wtlive

import (
	"regexp"
	"strings"
)

var hashtagRe = regexp.MustCompile(`class="WTL-Embed-Hashtag">(#\w+)</a>`)

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

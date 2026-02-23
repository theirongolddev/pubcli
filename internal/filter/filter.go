package filter

import (
	"html"
	"strings"

	"github.com/tayloree/publix-deals/internal/api"
)

// Options holds all filter criteria.
type Options struct {
	BOGO       bool
	Category   string
	Department string
	Query      string
	Limit      int
}

// Apply filters a slice of SavingItems according to the given options.
func Apply(items []api.SavingItem, opts Options) []api.SavingItem {
	result := items

	if opts.BOGO {
		result = where(result, func(i api.SavingItem) bool {
			return ContainsIgnoreCase(i.Categories, "bogo")
		})
	}

	if opts.Category != "" {
		result = where(result, func(i api.SavingItem) bool {
			return ContainsIgnoreCase(i.Categories, opts.Category)
		})
	}

	if opts.Department != "" {
		dept := strings.ToLower(opts.Department)
		result = where(result, func(i api.SavingItem) bool {
			return strings.Contains(strings.ToLower(Deref(i.Department)), dept)
		})
	}

	if opts.Query != "" {
		q := strings.ToLower(opts.Query)
		result = where(result, func(i api.SavingItem) bool {
			title := strings.ToLower(CleanText(Deref(i.Title)))
			desc := strings.ToLower(CleanText(Deref(i.Description)))
			return strings.Contains(title, q) || strings.Contains(desc, q)
		})
	}

	if opts.Limit > 0 && opts.Limit < len(result) {
		result = result[:opts.Limit]
	}

	return result
}

// Categories returns a map of category name to count across all items.
func Categories(items []api.SavingItem) map[string]int {
	cats := make(map[string]int)
	for _, item := range items {
		for _, c := range item.Categories {
			cats[c]++
		}
	}
	return cats
}

// Deref safely dereferences a string pointer, returning "" for nil.
func Deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// CleanText unescapes HTML entities and normalizes whitespace.
func CleanText(s string) string {
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func where(items []api.SavingItem, fn func(api.SavingItem) bool) []api.SavingItem {
	var result []api.SavingItem
	for _, item := range items {
		if fn(item) {
			result = append(result, item)
		}
	}
	return result
}

// ContainsIgnoreCase reports whether any element in slice matches val case-insensitively.
func ContainsIgnoreCase(slice []string, val string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, val) {
			return true
		}
	}
	return false
}

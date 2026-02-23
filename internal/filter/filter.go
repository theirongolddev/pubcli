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
	wantCategory := opts.Category != ""
	wantDepartment := opts.Department != ""
	wantQuery := opts.Query != ""
	needsFiltering := opts.BOGO || wantCategory || wantDepartment || wantQuery

	if !needsFiltering {
		if opts.Limit > 0 && opts.Limit < len(items) {
			return items[:opts.Limit]
		}
		return items
	}

	var result []api.SavingItem
	if opts.Limit > 0 && opts.Limit < len(items) {
		result = make([]api.SavingItem, 0, opts.Limit)
	} else {
		result = make([]api.SavingItem, 0, len(items))
	}

	category := opts.Category
	department := strings.ToLower(opts.Department)
	query := strings.ToLower(opts.Query)

	for _, item := range items {
		if opts.BOGO || wantCategory {
			hasBogo := !opts.BOGO
			hasCategory := !wantCategory

			for _, c := range item.Categories {
				if !hasBogo && strings.EqualFold(c, "bogo") {
					hasBogo = true
				}
				if !hasCategory && strings.EqualFold(c, category) {
					hasCategory = true
				}
				if hasBogo && hasCategory {
					break
				}
			}

			if !hasBogo || !hasCategory {
				continue
			}
		}

		if wantDepartment && !strings.Contains(strings.ToLower(Deref(item.Department)), department) {
			continue
		}

		if wantQuery {
			title := strings.ToLower(CleanText(Deref(item.Title)))
			desc := strings.ToLower(CleanText(Deref(item.Description)))
			if !strings.Contains(title, query) && !strings.Contains(desc, query) {
				continue
			}
		}

		result = append(result, item)
		if opts.Limit > 0 && len(result) >= opts.Limit {
			break
		}
	}

	if len(result) == 0 {
		return nil
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
	if !strings.ContainsAny(s, "&\r\n") {
		return strings.TrimSpace(s)
	}

	s = html.UnescapeString(s)
	if !strings.ContainsAny(s, "\r\n") {
		return strings.TrimSpace(s)
	}

	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
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

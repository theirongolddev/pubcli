package filter

import (
	"html"
	"sort"
	"strings"

	"github.com/tayloree/publix-deals/internal/api"
)

// Options holds all filter criteria.
type Options struct {
	BOGO       bool
	Category   string
	Department string
	Query      string
	Sort       string
	Limit      int
}

// Apply filters a slice of SavingItems according to the given options.
func Apply(items []api.SavingItem, opts Options) []api.SavingItem {
	wantCategory := opts.Category != ""
	wantDepartment := opts.Department != ""
	wantQuery := opts.Query != ""
	needsFiltering := opts.BOGO || wantCategory || wantDepartment || wantQuery
	sortMode := normalizeSortMode(opts.Sort)
	hasSort := sortMode != ""

	if !needsFiltering && !hasSort {
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

	department := strings.ToLower(opts.Department)
	query := strings.ToLower(opts.Query)
	applyLimitWhileFiltering := !hasSort && opts.Limit > 0
	categoryMatcher := newCategoryMatcher(opts.Category)

	for _, item := range items {
		if opts.BOGO || wantCategory {
			hasBogo := !opts.BOGO
			hasCategory := !wantCategory

			for _, c := range item.Categories {
				if !hasBogo && strings.EqualFold(c, "bogo") {
					hasBogo = true
				}
				if !hasCategory && categoryMatcher.matches(c) {
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
		if applyLimitWhileFiltering && len(result) >= opts.Limit {
			break
		}
	}

	if hasSort && len(result) > 1 {
		sortItems(result, sortMode)
	}
	if opts.Limit > 0 && opts.Limit < len(result) {
		result = result[:opts.Limit]
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

func sortItems(items []api.SavingItem, mode string) {
	switch mode {
	case "savings":
		sort.SliceStable(items, func(i, j int) bool {
			left := DealScore(items[i])
			right := DealScore(items[j])
			if left == right {
				return strings.ToLower(CleanText(Deref(items[i].Title))) < strings.ToLower(CleanText(Deref(items[j].Title)))
			}
			return left > right
		})
	case "ending":
		sort.SliceStable(items, func(i, j int) bool {
			leftDate, leftOK := parseDealDate(items[i].EndFormatted)
			rightDate, rightOK := parseDealDate(items[j].EndFormatted)
			switch {
			case leftOK && rightOK:
				if leftDate.Equal(rightDate) {
					return DealScore(items[i]) > DealScore(items[j])
				}
				return leftDate.Before(rightDate)
			case leftOK:
				return true
			case rightOK:
				return false
			default:
				return DealScore(items[i]) > DealScore(items[j])
			}
		})
	}
}

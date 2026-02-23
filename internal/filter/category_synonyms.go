package filter

import "strings"

var categorySynonyms = map[string][]string{
	"bogo":    {"bogof", "buy one get one", "buy1get1", "2 for 1", "two for one"},
	"produce": {"fruit", "fruits", "vegetable", "vegetables", "veggie", "veggies"},
	"meat":    {"beef", "chicken", "poultry", "pork", "seafood"},
	"dairy":   {"milk", "cheese", "yogurt"},
	"bakery":  {"bread", "pastry", "pastries"},
	"deli":    {"delicatessen", "cold cuts", "lunch meat"},
	"frozen":  {"frozen foods"},
	"grocery": {"pantry", "shelf"},
}

type categoryMatcher struct {
	exactAliases []string
	normalized   map[string]struct{}
}

func newCategoryMatcher(wanted string) categoryMatcher {
	aliases := categoryAliasList(wanted)
	if len(aliases) == 0 {
		return categoryMatcher{}
	}

	normalized := make(map[string]struct{}, len(aliases))
	for _, alias := range aliases {
		normalized[normalizeCategory(alias)] = struct{}{}
	}

	return categoryMatcher{
		exactAliases: aliases,
		normalized:   normalized,
	}
}

func categoryAliasList(wanted string) []string {
	raw := strings.TrimSpace(wanted)
	group := resolveCategoryGroup(wanted)
	if raw == "" && group == "" {
		return nil
	}

	out := make([]string, 0, 1+len(categorySynonyms[group]))
	addAlias := func(alias string) {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			return
		}
		for _, existing := range out {
			if strings.EqualFold(existing, alias) {
				return
			}
		}
		out = append(out, alias)
	}

	addAlias(raw)
	addAlias(group)

	if synonyms, ok := categorySynonyms[group]; ok {
		out = append(out, synonyms...)
	}
	return out
}

func resolveCategoryGroup(wanted string) string {
	norm := normalizeCategory(wanted)
	if norm == "" {
		return ""
	}

	if _, ok := categorySynonyms[norm]; ok {
		return norm
	}
	for key, synonyms := range categorySynonyms {
		for _, s := range synonyms {
			if normalizeCategory(s) == norm {
				return key
			}
		}
	}
	return norm
}

func (m categoryMatcher) matches(category string) bool {
	trimmed := strings.TrimSpace(category)
	for _, alias := range m.exactAliases {
		if strings.EqualFold(trimmed, alias) {
			return true
		}
	}

	// Fast path: if no separators are present and direct aliases didn't match,
	// normalization would only add overhead for common categories like "grocery".
	if !strings.ContainsAny(trimmed, "-_ ") {
		return false
	}

	norm := normalizeCategory(trimmed)
	_, ok := m.normalized[norm]
	return ok
}

func normalizeCategory(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.Join(strings.Fields(s), " ")
	switch {
	case len(s) > 4 && strings.HasSuffix(s, "ies"):
		s = strings.TrimSuffix(s, "ies") + "y"
	case len(s) > 3 && strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss"):
		s = strings.TrimSuffix(s, "s")
	}
	return s
}

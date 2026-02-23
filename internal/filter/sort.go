package filter

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tayloree/publix-deals/internal/api"
)

var (
	reDollar  = regexp.MustCompile(`\$(\d+(?:\.\d{1,2})?)`)
	rePercent = regexp.MustCompile(`(\d{1,3})\s*%`)
)

// DealScore estimates relative deal value for ranking.
func DealScore(item api.SavingItem) float64 {
	score := 0.0

	if ContainsIgnoreCase(item.Categories, "bogo") {
		score += 8
	}

	text := strings.ToLower(
		CleanText(Deref(item.Savings) + " " + Deref(item.AdditionalDealInfo)),
	)
	for _, m := range reDollar.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 {
			continue
		}
		if amount, err := strconv.ParseFloat(m[1], 64); err == nil {
			score += amount
		}
	}
	for _, m := range rePercent.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 {
			continue
		}
		if pct, err := strconv.ParseFloat(m[1], 64); err == nil {
			score += pct / 20.0
		}
	}

	if score == 0 {
		return 0.01
	}
	return score
}

func normalizeSortMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "relevance":
		return ""
	case "savings":
		return "savings"
	case "ending", "end", "expiry", "expiration":
		return "ending"
	default:
		return ""
	}
}

func parseDealDate(raw string) (time.Time, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, false
	}

	layouts := []string{
		"1/2/2006",
		"01/02/2006",
		"1/2/06",
		"01/02/06",
		"2006-01-02",
		"Jan 2, 2006",
		"January 2, 2006",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

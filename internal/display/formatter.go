package display

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tayloree/publix-deals/internal/api"
	"github.com/tayloree/publix-deals/internal/filter"
)

// Styles for terminal output.
var (
	titleStyle   = lipgloss.NewStyle().Bold(true)
	bogoTag      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5")) // magenta
	priceStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))            // green
	dealStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))            // yellow
	dimStyle     = lipgloss.NewStyle().Faint(true)
	cyanStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)

// DealJSON is the JSON output shape for a deal.
type DealJSON struct {
	Title       string   `json:"title"`
	Savings     string   `json:"savings"`
	Description string   `json:"description"`
	Department  string   `json:"department"`
	Categories  []string `json:"categories"`
	DealInfo    string   `json:"additionalDealInfo"`
	Brand       string   `json:"brand"`
	ValidFrom   string   `json:"validFrom"`
	ValidTo     string   `json:"validTo"`
	IsBogo      bool     `json:"isBogo"`
	ImageURL    string   `json:"imageUrl"`
}

// StoreJSON is the JSON output shape for a store.
type StoreJSON struct {
	Number   string `json:"number"`
	Name     string `json:"name"`
	Address  string `json:"address"`
	Distance string `json:"distance"`
}

// PrintDeals renders a list of deals to the writer.
func PrintDeals(w io.Writer, items []api.SavingItem) {
	dateRange := ""
	if len(items) > 0 && items[0].StartFormatted != "" {
		dateRange = fmt.Sprintf(" (%s - %s)", items[0].StartFormatted, items[0].EndFormatted)
	}

	fmt.Fprintf(w, "\n%s%s — %s\n\n",
		headerStyle.Render("Publix Weekly Deals"),
		dateRange,
		cyanStyle.Render(fmt.Sprintf("%d items", len(items))),
	)

	for _, item := range items {
		printDeal(w, item)
		fmt.Fprintln(w)
	}
}

// PrintDealsJSON renders deals as JSON.
func PrintDealsJSON(w io.Writer, items []api.SavingItem) error {
	out := make([]DealJSON, 0, len(items))
	for _, item := range items {
		out = append(out, toDealJSON(item))
	}
	return json.NewEncoder(w).Encode(out)
}

// PrintStores renders a list of stores to the writer.
func PrintStores(w io.Writer, stores []api.Store, zipCode string) {
	fmt.Fprintf(w, "\n%s\n\n",
		titleStyle.Render(fmt.Sprintf("Publix stores near %s:", zipCode)),
	)
	for _, s := range stores {
		num := api.StoreNumber(s.Key)
		fmt.Fprintf(w, "  %s  %s\n", cyanStyle.Render("#"+num), titleStyle.Render(s.Name))
		fmt.Fprintf(w, "        %s, %s, %s %s\n", s.Addr, s.City, s.State, s.Zip)
		if s.Distance != "" {
			fmt.Fprintf(w, "        %s\n", dimStyle.Render(s.Distance+" miles"))
		}
		fmt.Fprintln(w)
	}
}

// PrintStoresJSON renders stores as JSON.
func PrintStoresJSON(w io.Writer, stores []api.Store) error {
	out := make([]StoreJSON, 0, len(stores))
	for _, s := range stores {
		out = append(out, StoreJSON{
			Number:   api.StoreNumber(s.Key),
			Name:     s.Name,
			Address:  fmt.Sprintf("%s, %s, %s %s", s.Addr, s.City, s.State, s.Zip),
			Distance: s.Distance,
		})
	}
	return json.NewEncoder(w).Encode(out)
}

// PrintCategories renders a list of categories and their counts.
func PrintCategories(w io.Writer, cats map[string]int, storeNumber string) {
	type catCount struct {
		Name  string
		Count int
	}
	sorted := make([]catCount, 0, len(cats))
	for k, v := range cats {
		sorted = append(sorted, catCount{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Count > sorted[j].Count })

	fmt.Fprintf(w, "\n%s\n\n",
		titleStyle.Render(fmt.Sprintf("Categories for store #%s this week:", storeNumber)),
	)
	for _, c := range sorted {
		fmt.Fprintf(w, "  %s: %d deals\n", cyanStyle.Render(c.Name), c.Count)
	}
	fmt.Fprintln(w)
}

// PrintCategoriesJSON renders categories as JSON.
func PrintCategoriesJSON(w io.Writer, cats map[string]int) error {
	return json.NewEncoder(w).Encode(cats)
}

// PrintStoreContext prints a dim line showing which store was auto-selected.
func PrintStoreContext(w io.Writer, store api.Store) {
	num := api.StoreNumber(store.Key)
	fmt.Fprintf(w, "%s\n\n",
		dimStyle.Render(fmt.Sprintf("Using store: #%s — %s (%s, %s)", num, store.Name, store.City, store.State)),
	)
}

// PrintError prints a styled error message.
func PrintError(w io.Writer, msg string) {
	fmt.Fprintln(w, errorStyle.Render(msg))
}

// PrintWarning prints a styled warning message.
func PrintWarning(w io.Writer, msg string) {
	fmt.Fprintln(w, warningStyle.Render(msg))
}

func printDeal(w io.Writer, item api.SavingItem) {
	title := filter.CleanText(filter.Deref(item.Title))
	if title == "" {
		title = "Unknown"
	}
	savings := filter.CleanText(filter.Deref(item.Savings))
	desc := filter.CleanText(filter.Deref(item.Description))
	dept := filter.CleanText(filter.Deref(item.Department))
	dealInfo := filter.CleanText(filter.Deref(item.AdditionalDealInfo))
	isBogo := filter.ContainsIgnoreCase(item.Categories, "bogo")

	// Title line
	tag := ""
	if isBogo {
		tag = bogoTag.Render("BOGO") + " "
	}
	fmt.Fprintf(w, "  %s%s\n", tag, titleStyle.Render(title))

	// Price / savings
	var parts []string
	if savings != "" {
		parts = append(parts, priceStyle.Render(savings))
	}
	if dealInfo != "" {
		parts = append(parts, dealStyle.Render(dealInfo))
	}
	if len(parts) > 0 {
		fmt.Fprintf(w, "    %s\n", strings.Join(parts, " | "))
	}

	// Description
	if desc != "" {
		fmt.Fprintf(w, "    %s\n", dimStyle.Render(wordWrap(desc, 72, "    ")))
	}

	// Meta
	var meta []string
	if item.StartFormatted != "" && item.EndFormatted != "" {
		meta = append(meta, fmt.Sprintf("Valid %s - %s", item.StartFormatted, item.EndFormatted))
	}
	if dept != "" {
		meta = append(meta, dept)
	}
	if len(meta) > 0 {
		fmt.Fprintf(w, "    %s\n", dimStyle.Render(strings.Join(meta, " | ")))
	}
}

func toDealJSON(item api.SavingItem) DealJSON {
	categories := item.Categories
	if categories == nil {
		categories = []string{}
	}
	return DealJSON{
		Title:       filter.CleanText(filter.Deref(item.Title)),
		Savings:     filter.CleanText(filter.Deref(item.Savings)),
		Description: filter.CleanText(filter.Deref(item.Description)),
		Department:  filter.CleanText(filter.Deref(item.Department)),
		Categories:  categories,
		DealInfo:    filter.CleanText(filter.Deref(item.AdditionalDealInfo)),
		Brand:       filter.CleanText(filter.Deref(item.Brand)),
		ValidFrom:   item.StartFormatted,
		ValidTo:     item.EndFormatted,
		IsBogo:      filter.ContainsIgnoreCase(item.Categories, "bogo"),
		ImageURL:    filter.Deref(item.ImageURL),
	}
}

func wordWrap(text string, width int, indent string) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	line := words[0]
	for _, w := range words[1:] {
		if len(line)+1+len(w) > width {
			lines = append(lines, line)
			line = w
		} else {
			line += " " + w
		}
	}
	lines = append(lines, line)
	return strings.Join(lines, "\n"+indent)
}

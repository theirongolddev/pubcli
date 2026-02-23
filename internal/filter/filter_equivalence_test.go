package filter_test

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tayloree/publix-deals/internal/api"
	"github.com/tayloree/publix-deals/internal/filter"
)

func referenceApply(items []api.SavingItem, opts filter.Options) []api.SavingItem {
	result := items

	if opts.BOGO {
		result = referenceWhere(result, func(i api.SavingItem) bool {
			return filter.ContainsIgnoreCase(i.Categories, "bogo")
		})
	}

	if opts.Category != "" {
		result = referenceWhere(result, func(i api.SavingItem) bool {
			return filter.ContainsIgnoreCase(i.Categories, opts.Category)
		})
	}

	if opts.Department != "" {
		dept := strings.ToLower(opts.Department)
		result = referenceWhere(result, func(i api.SavingItem) bool {
			return strings.Contains(strings.ToLower(filter.Deref(i.Department)), dept)
		})
	}

	if opts.Query != "" {
		q := strings.ToLower(opts.Query)
		result = referenceWhere(result, func(i api.SavingItem) bool {
			title := strings.ToLower(filter.CleanText(filter.Deref(i.Title)))
			desc := strings.ToLower(filter.CleanText(filter.Deref(i.Description)))
			return strings.Contains(title, q) || strings.Contains(desc, q)
		})
	}

	if opts.Limit > 0 && opts.Limit < len(result) {
		result = result[:opts.Limit]
	}

	return result
}

func referenceWhere(items []api.SavingItem, fn func(api.SavingItem) bool) []api.SavingItem {
	var result []api.SavingItem
	for _, item := range items {
		if fn(item) {
			result = append(result, item)
		}
	}
	return result
}

func randomItem(rng *rand.Rand, idx int) api.SavingItem {
	makePtr := func(v string) *string { return &v }

	var title *string
	if rng.Intn(4) != 0 {
		title = makePtr(fmt.Sprintf("Fresh Deal %d", idx))
	}

	var desc *string
	if rng.Intn(3) != 0 {
		desc = makePtr(fmt.Sprintf("Weekly offer %d", idx))
	}

	var dept *string
	deptOptions := []*string{
		nil,
		makePtr("Grocery"),
		makePtr("Produce"),
		makePtr("Meat"),
		makePtr("Frozen"),
	}
	dept = deptOptions[rng.Intn(len(deptOptions))]

	catPool := []string{"bogo", "grocery", "produce", "meat", "frozen", "dairy"}
	catCount := rng.Intn(4)
	cats := make([]string, 0, catCount)
	for range catCount {
		cats = append(cats, catPool[rng.Intn(len(catPool))])
	}

	return api.SavingItem{
		ID:          fmt.Sprintf("id-%d", idx),
		Title:       title,
		Description: desc,
		Department:  dept,
		Categories:  cats,
	}
}

func randomOptions(rng *rand.Rand) filter.Options {
	categories := []string{"", "bogo", "grocery", "produce", "meat"}
	departments := []string{"", "groc", "prod", "meat"}
	queries := []string{"", "fresh", "offer", "deal"}
	limits := []int{0, 1, 3, 5, 10}
	return filter.Options{
		BOGO:       rng.Intn(2) == 0,
		Category:   categories[rng.Intn(len(categories))],
		Department: departments[rng.Intn(len(departments))],
		Query:      queries[rng.Intn(len(queries))],
		Limit:      limits[rng.Intn(len(limits))],
	}
}

func TestApply_ReferenceEquivalence(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	for caseNum := 0; caseNum < 500; caseNum++ {
		itemCount := rng.Intn(60)
		items := make([]api.SavingItem, 0, itemCount)
		for i := range itemCount {
			items = append(items, randomItem(rng, i))
		}

		opts := randomOptions(rng)
		got := filter.Apply(items, opts)
		want := referenceApply(items, opts)

		assert.Equal(t, want, got, "mismatch for opts=%+v case=%d", opts, caseNum)
	}
}

func BenchmarkApply_ReferenceWorkload_1kDeals(b *testing.B) {
	rng := rand.New(rand.NewSource(7))
	items := make([]api.SavingItem, 0, 1000)
	for i := 0; i < 1000; i++ {
		items = append(items, randomItem(rng, i))
	}
	opts := filter.Options{
		BOGO:       true,
		Category:   "grocery",
		Department: "groc",
		Query:      "deal",
		Limit:      50,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = filter.Apply(items, opts)
	}
}

func TestApply_AllocationBudget(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	items := make([]api.SavingItem, 0, 1000)
	for i := 0; i < 1000; i++ {
		items = append(items, randomItem(rng, i))
	}
	opts := filter.Options{
		BOGO:       true,
		Category:   "grocery",
		Department: "groc",
		Query:      "deal",
		Limit:      50,
	}

	allocs := testing.AllocsPerRun(100, func() {
		_ = filter.Apply(items, opts)
	})

	// Guardrail for accidental reintroduction of multi-pass intermediate slices.
	assert.LessOrEqual(t, allocs, 80.0)
}

func BenchmarkApply_LegacyReference_1kDeals(b *testing.B) {
	rng := rand.New(rand.NewSource(7))
	items := make([]api.SavingItem, 0, 1000)
	for i := 0; i < 1000; i++ {
		items = append(items, randomItem(rng, i))
	}
	opts := filter.Options{
		BOGO:       true,
		Category:   "grocery",
		Department: "groc",
		Query:      "deal",
		Limit:      50,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = referenceApply(items, opts)
	}
}

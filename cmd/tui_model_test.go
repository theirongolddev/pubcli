package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tayloree/publix-deals/internal/api"
)

func strPtr(value string) *string { return &value }

func TestCanonicalSortMode(t *testing.T) {
	assert.Equal(t, "savings", canonicalSortMode("savings"))
	assert.Equal(t, "ending", canonicalSortMode("end"))
	assert.Equal(t, "ending", canonicalSortMode("expiry"))
	assert.Equal(t, "ending", canonicalSortMode("expiration"))
	assert.Equal(t, "", canonicalSortMode("relevance"))
	assert.Equal(t, "", canonicalSortMode("unknown"))
}

func TestBuildGroupedListItems_BogoFirstAndNumberedHeaders(t *testing.T) {
	deals := []api.SavingItem{
		{ID: "1", Title: strPtr("Bananas"), Categories: []string{"produce"}},
		{ID: "2", Title: strPtr("Chicken"), Categories: []string{"meat", "bogo"}},
		{ID: "3", Title: strPtr("Apples"), Categories: []string{"produce"}},
		{ID: "4", Title: strPtr("Ground Beef"), Categories: []string{"meat"}},
	}

	items, starts := buildGroupedListItems(deals)

	assert.NotEmpty(t, items)
	assert.Equal(t, []int{0, 2, 5}, starts)

	header, ok := items[0].(tuiGroupItem)
	assert.True(t, ok)
	assert.Equal(t, "BOGO", header.name)
	assert.Equal(t, 1, header.ordinal)

	header2, ok := items[2].(tuiGroupItem)
	assert.True(t, ok)
	assert.Equal(t, "Produce", header2.name)
	assert.Equal(t, 2, header2.count)

	header3, ok := items[5].(tuiGroupItem)
	assert.True(t, ok)
	assert.Equal(t, "Meat", header3.name)
	assert.Equal(t, 1, header3.count)
}

func TestBuildCategoryChoices_AlwaysIncludesCurrent(t *testing.T) {
	deals := []api.SavingItem{
		{Categories: []string{"produce"}},
		{Categories: []string{"meat"}},
	}

	choices := buildCategoryChoices(deals, "seafood")

	assert.Contains(t, choices, "")
	assert.Contains(t, choices, "produce")
	assert.Contains(t, choices, "meat")
	assert.Contains(t, choices, "seafood")
}

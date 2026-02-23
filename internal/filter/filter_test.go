package filter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tayloree/publix-deals/internal/api"
	"github.com/tayloree/publix-deals/internal/filter"
)

func ptr(s string) *string { return &s }

func sampleItems() []api.SavingItem {
	return []api.SavingItem{
		{
			ID:         "1",
			Title:      ptr("Chicken Breasts"),
			Department: ptr("Meat"),
			Categories: []string{"meat"},
		},
		{
			ID:         "2",
			Title:      ptr("Nutella Spread"),
			Savings:    ptr("Buy 1 Get 1 FREE"),
			Department: ptr("Peanut Butter & Jelly"),
			Categories: []string{"bogo", "grocery"},
		},
		{
			ID:          "3",
			Title:       ptr("Organic Spinach"),
			Description: ptr("Fresh baby spinach, 5-oz pkg."),
			Department:  ptr("Produce"),
			Categories:  []string{"produce"},
		},
		{
			ID:         "4",
			Title:      ptr("Dog Food"),
			Department: ptr("Pet Food"),
			Categories: []string{"bogo", "pet", "pet-bogos"},
		},
		{
			ID:         "5",
			Title:      nil,
			Department: nil,
			Categories: nil,
		},
	}
}

func TestApply_NoFilters(t *testing.T) {
	items := sampleItems()
	result := filter.Apply(items, filter.Options{})
	assert.Len(t, result, 5)
}

func TestApply_BOGO(t *testing.T) {
	result := filter.Apply(sampleItems(), filter.Options{BOGO: true})
	assert.Len(t, result, 2)
	assert.Equal(t, "2", result[0].ID)
	assert.Equal(t, "4", result[1].ID)
}

func TestApply_Category(t *testing.T) {
	result := filter.Apply(sampleItems(), filter.Options{Category: "meat"})
	assert.Len(t, result, 1)
	assert.Equal(t, "1", result[0].ID)
}

func TestApply_CategoryCaseInsensitive(t *testing.T) {
	result := filter.Apply(sampleItems(), filter.Options{Category: "BOGO"})
	assert.Len(t, result, 2)
}

func TestApply_Department(t *testing.T) {
	result := filter.Apply(sampleItems(), filter.Options{Department: "produce"})
	assert.Len(t, result, 1)
	assert.Equal(t, "Organic Spinach", *result[0].Title)
}

func TestApply_DepartmentPartialMatch(t *testing.T) {
	result := filter.Apply(sampleItems(), filter.Options{Department: "pet"})
	assert.Len(t, result, 1)
	assert.Equal(t, "4", result[0].ID)
}

func TestApply_Query(t *testing.T) {
	result := filter.Apply(sampleItems(), filter.Options{Query: "chicken"})
	assert.Len(t, result, 1)
	assert.Equal(t, "1", result[0].ID)
}

func TestApply_QueryMatchesDescription(t *testing.T) {
	result := filter.Apply(sampleItems(), filter.Options{Query: "spinach"})
	assert.Len(t, result, 1)
	assert.Equal(t, "3", result[0].ID)
}

func TestApply_QueryNoMatch(t *testing.T) {
	result := filter.Apply(sampleItems(), filter.Options{Query: "xyz123"})
	assert.Empty(t, result)
}

func TestApply_Limit(t *testing.T) {
	result := filter.Apply(sampleItems(), filter.Options{Limit: 2})
	assert.Len(t, result, 2)
}

func TestApply_CombinedFilters(t *testing.T) {
	result := filter.Apply(sampleItems(), filter.Options{
		BOGO:  true,
		Limit: 1,
	})
	assert.Len(t, result, 1)
	assert.Equal(t, "2", result[0].ID)
}

func TestApply_NilFields(t *testing.T) {
	// Item 5 has nil title/department/categories — should not panic
	result := filter.Apply(sampleItems(), filter.Options{Query: "anything"})
	assert.Empty(t, result)
}

func TestCategories(t *testing.T) {
	cats := filter.Categories(sampleItems())

	assert.Equal(t, 2, cats["bogo"])
	assert.Equal(t, 1, cats["meat"])
	assert.Equal(t, 1, cats["grocery"])
	assert.Equal(t, 1, cats["produce"])
	assert.Equal(t, 1, cats["pet"])
	assert.Equal(t, 1, cats["pet-bogos"])
}

func TestDeref(t *testing.T) {
	s := "hello"
	assert.Equal(t, "hello", filter.Deref(&s))
	assert.Equal(t, "", filter.Deref(nil))
}

func TestCleanText(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello &amp; World", "Hello & World"},
		{"Line1\r\nLine2", "Line1 Line2"},
		{"  spaces  ", "spaces"},
		{"Eight O&#39;Clock", "Eight O'Clock"},
		{"", ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, filter.CleanText(tt.input), "CleanText(%q)", tt.input)
	}
}

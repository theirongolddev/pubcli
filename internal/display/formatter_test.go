package display_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tayloree/publix-deals/internal/api"
	"github.com/tayloree/publix-deals/internal/display"
)

func ptr(s string) *string { return &s }

func sampleDeals() []api.SavingItem {
	return []api.SavingItem{
		{
			ID:                 "1",
			Title:              ptr("Chicken Breasts"),
			Savings:            ptr("$3.99 lb"),
			Description:        ptr("USDA Grade A, 97% Fat Free"),
			Department:         ptr("Meat"),
			Brand:              ptr("Publix"),
			Categories:         []string{"meat"},
			AdditionalDealInfo: ptr("SAVE UP TO $1.00 LB"),
			StartFormatted:     "2/18",
			EndFormatted:       "2/24",
		},
		{
			ID:             "2",
			Title:          ptr("Nutella &amp; More"),
			Savings:        ptr("Buy 1 Get 1 FREE"),
			Department:     ptr("Grocery"),
			Categories:     []string{"bogo", "grocery"},
			StartFormatted: "2/18",
			EndFormatted:   "2/24",
		},
	}
}

func TestPrintDeals_ContainsExpectedContent(t *testing.T) {
	var buf bytes.Buffer
	display.PrintDeals(&buf, sampleDeals())
	output := buf.String()

	assert.Contains(t, output, "Publix Weekly Deals")
	assert.Contains(t, output, "2 items")
	assert.Contains(t, output, "Chicken Breasts")
	assert.Contains(t, output, "$3.99 lb")
	assert.Contains(t, output, "SAVE UP TO $1.00 LB")
	assert.Contains(t, output, "BOGO") // Nutella is bogo
	// HTML entities should be unescaped
	assert.Contains(t, output, "Nutella & More")
	assert.NotContains(t, output, "&amp;")
}

func TestPrintDeals_FallbackTitleFromBrandAndDepartment(t *testing.T) {
	items := []api.SavingItem{
		{
			ID:         "fallback-1",
			Title:      nil,
			Brand:      ptr("Publix"),
			Department: ptr("Meat"),
			Categories: []string{"meat"},
		},
	}

	var buf bytes.Buffer
	display.PrintDeals(&buf, items)
	output := buf.String()

	assert.Contains(t, output, "Publix deal (Meat)")
	assert.NotContains(t, output, "Unknown")
}

func TestPrintDeals_FallbackTitleFromID(t *testing.T) {
	items := []api.SavingItem{
		{
			ID:    "fallback-2",
			Title: nil,
		},
	}

	var buf bytes.Buffer
	display.PrintDeals(&buf, items)
	output := buf.String()

	assert.Contains(t, output, "Deal fallback-2")
	assert.NotContains(t, output, "Unknown")
}

func TestPrintDealsJSON(t *testing.T) {
	var buf bytes.Buffer
	err := display.PrintDealsJSON(&buf, sampleDeals())
	require.NoError(t, err)
	assert.NotContains(t, buf.String(), "\n  ")

	var deals []display.DealJSON
	err = json.Unmarshal(buf.Bytes(), &deals)
	require.NoError(t, err)

	assert.Len(t, deals, 2)
	assert.Equal(t, "Chicken Breasts", deals[0].Title)
	assert.Equal(t, "$3.99 lb", deals[0].Savings)
	assert.Equal(t, "Meat", deals[0].Department)
	assert.False(t, deals[0].IsBogo)

	// HTML entities should be clean in JSON too
	assert.Equal(t, "Nutella & More", deals[1].Title)
	assert.True(t, deals[1].IsBogo)
}

func TestPrintDealsJSON_NilFields(t *testing.T) {
	items := []api.SavingItem{{ID: "nil-test"}}
	var buf bytes.Buffer
	err := display.PrintDealsJSON(&buf, items)
	require.NoError(t, err)

	var deals []display.DealJSON
	err = json.Unmarshal(buf.Bytes(), &deals)
	require.NoError(t, err)
	assert.Len(t, deals, 1)
	assert.Equal(t, "", deals[0].Title)
	assert.NotNil(t, deals[0].Categories)
}

func TestPrintStores(t *testing.T) {
	stores := []api.Store{
		{Key: "01425", Name: "Peachers Mill", Addr: "1490 Tiny Town Rd", City: "Clarksville", State: "TN", Zip: "37042", Distance: "5"},
	}
	var buf bytes.Buffer
	display.PrintStores(&buf, stores, "37042")
	output := buf.String()

	assert.Contains(t, output, "37042")
	assert.Contains(t, output, "#1425")
	assert.Contains(t, output, "Peachers Mill")
	assert.Contains(t, output, "5 miles")
}

func TestPrintStoresJSON(t *testing.T) {
	stores := []api.Store{
		{Key: "01425", Name: "Peachers Mill", Addr: "1490 Tiny Town Rd", City: "Clarksville", State: "TN", Zip: "37042", Distance: "5"},
	}
	var buf bytes.Buffer
	err := display.PrintStoresJSON(&buf, stores)
	require.NoError(t, err)
	assert.NotContains(t, buf.String(), "\n  ")

	var out []display.StoreJSON
	err = json.Unmarshal(buf.Bytes(), &out)
	require.NoError(t, err)

	assert.Len(t, out, 1)
	assert.Equal(t, "1425", out[0].Number)
	assert.Equal(t, "Peachers Mill", out[0].Name)
	assert.Contains(t, out[0].Address, "Clarksville")
}

func TestPrintCategories(t *testing.T) {
	cats := map[string]int{"bogo": 10, "meat": 5, "produce": 3}
	var buf bytes.Buffer
	display.PrintCategories(&buf, cats, "1425")
	output := buf.String()

	assert.Contains(t, output, "1425")
	assert.Contains(t, output, "bogo")
	assert.Contains(t, output, "10 deals")
	assert.Contains(t, output, "meat")
	assert.Contains(t, output, "produce")
}

func TestPrintCategoriesJSON(t *testing.T) {
	cats := map[string]int{"bogo": 10, "meat": 5}
	var buf bytes.Buffer
	err := display.PrintCategoriesJSON(&buf, cats)
	require.NoError(t, err)
	assert.NotContains(t, buf.String(), "\n  ")

	var out map[string]int
	err = json.Unmarshal(buf.Bytes(), &out)
	require.NoError(t, err)

	assert.Equal(t, 10, out["bogo"])
	assert.Equal(t, 5, out["meat"])
}

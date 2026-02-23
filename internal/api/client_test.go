package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tayloree/publix-deals/internal/api"
)

func ptr(s string) *string { return &s }

func newTestSavingsServer(t *testing.T, storeNumber string, items []api.SavingItem) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the PublixStore header is sent
		got := r.Header.Get("PublixStore")
		if storeNumber != "" {
			assert.Equal(t, storeNumber, got, "PublixStore header mismatch")
		}

		resp := api.SavingsResponse{
			Savings:    items,
			LanguageID: 1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func newTestStoreServer(t *testing.T, stores []api.Store) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.URL.Query().Get("zipCode"), "zipCode param required")

		resp := api.StoreResponse{Stores: stores}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestFetchSavings(t *testing.T) {
	items := []api.SavingItem{
		{
			ID:             "test-1",
			Title:          ptr("Chicken Breasts"),
			Savings:        ptr("$3.99 lb"),
			Department:     ptr("Meat"),
			Categories:     []string{"meat"},
			StartFormatted: "2/18",
			EndFormatted:   "2/24",
		},
		{
			ID:         "test-2",
			Title:      ptr("Nutella"),
			Savings:    ptr("Buy 1 Get 1 FREE"),
			Categories: []string{"bogo", "grocery"},
		},
	}

	srv := newTestSavingsServer(t, "1425", items)
	defer srv.Close()

	client := api.NewClientWithBaseURLs(srv.URL, "")
	resp, err := client.FetchSavings(context.Background(), "1425")

	require.NoError(t, err)
	assert.Len(t, resp.Savings, 2)
	assert.Equal(t, "Chicken Breasts", *resp.Savings[0].Title)
	assert.Equal(t, "Buy 1 Get 1 FREE", *resp.Savings[1].Savings)
}

func TestFetchSavings_EmptyStore(t *testing.T) {
	srv := newTestSavingsServer(t, "", nil)
	defer srv.Close()

	client := api.NewClientWithBaseURLs(srv.URL, "")
	resp, err := client.FetchSavings(context.Background(), "")

	require.NoError(t, err)
	assert.Empty(t, resp.Savings)
}

func TestFetchSavings_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := api.NewClientWithBaseURLs(srv.URL, "")
	_, err := client.FetchSavings(context.Background(), "1425")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestFetchStores(t *testing.T) {
	stores := []api.Store{
		{Key: "01425", Name: "Peachers Mill", City: "Clarksville", State: "TN", Zip: "37042", Distance: "5"},
		{Key: "00100", Name: "Downtown", City: "Nashville", State: "TN", Zip: "37201", Distance: "15"},
	}

	srv := newTestStoreServer(t, stores)
	defer srv.Close()

	client := api.NewClientWithBaseURLs("", srv.URL)
	result, err := client.FetchStores(context.Background(), "37042", 5)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "Peachers Mill", result[0].Name)
	assert.Equal(t, "01425", result[0].Key)
}

func TestFetchStores_NoResults(t *testing.T) {
	srv := newTestStoreServer(t, nil)
	defer srv.Close()

	client := api.NewClientWithBaseURLs("", srv.URL)
	result, err := client.FetchStores(context.Background(), "00000", 5)

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestStoreNumber(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"01425", "1425"},
		{"00100", "100"},
		{"1425", "1425"},
		{"0", ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, api.StoreNumber(tt.input), "StoreNumber(%q)", tt.input)
	}
}

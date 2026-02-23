package perf_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tayloree/publix-deals/internal/api"
	"github.com/tayloree/publix-deals/internal/display"
	"github.com/tayloree/publix-deals/internal/filter"
)

func strPtr(v string) *string { return &v }

func benchmarkDeals(count int) []api.SavingItem {
	items := make([]api.SavingItem, 0, count)
	for i := range count {
		title := fmt.Sprintf("Fresh item %d", i)
		desc := fmt.Sprintf("Fresh weekly deal %d with great savings", i)
		savings := fmt.Sprintf("$%d.99", (i%9)+1)
		dept := "Grocery"
		if i%4 == 0 {
			dept = "Produce"
		}
		if i%7 == 0 {
			dept = "Meat"
		}
		cats := []string{"grocery"}
		if i%3 == 0 {
			cats = append(cats, "bogo")
		}
		if i%5 == 0 {
			cats = append(cats, "produce")
		}
		if i%7 == 0 {
			cats = append(cats, "meat")
		}
		items = append(items, api.SavingItem{
			ID:             fmt.Sprintf("id-%d", i),
			Title:          strPtr(title),
			Description:    strPtr(desc),
			Savings:        strPtr(savings),
			Department:     strPtr(dept),
			Categories:     cats,
			StartFormatted: "2/18",
			EndFormatted:   "2/24",
		})
	}
	return items
}

func setupPipelineServer(b *testing.B, dealCount int) (*httptest.Server, *api.Client) {
	b.Helper()

	storesPayload, err := json.Marshal(api.StoreResponse{
		Stores: []api.Store{
			{Key: "01425", Name: "Peachers Mill", Addr: "1490 Tiny Town Rd", City: "Clarksville", State: "TN", Zip: "37042", Distance: "5"},
		},
	})
	if err != nil {
		b.Fatalf("marshal stores payload: %v", err)
	}

	savingsPayload, err := json.Marshal(api.SavingsResponse{
		Savings:    benchmarkDeals(dealCount),
		LanguageID: 1,
	})
	if err != nil {
		b.Fatalf("marshal savings payload: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/stores":
			_, _ = w.Write(storesPayload)
		case "/savings":
			_, _ = w.Write(savingsPayload)
		default:
			http.NotFound(w, r)
		}
	}))
	b.Cleanup(server.Close)

	client := api.NewClientWithBaseURLs(server.URL+"/savings", server.URL+"/stores")
	return server, client
}

func runPipeline(b *testing.B, client *api.Client) {
	b.Helper()

	ctx := context.Background()
	stores, err := client.FetchStores(ctx, "33101", 1)
	if err != nil {
		b.Fatalf("fetch stores: %v", err)
	}
	if len(stores) == 0 {
		b.Fatalf("fetch stores: empty result")
	}

	resp, err := client.FetchSavings(ctx, api.StoreNumber(stores[0].Key))
	if err != nil {
		b.Fatalf("fetch savings: %v", err)
	}

	filtered := filter.Apply(resp.Savings, filter.Options{
		BOGO:       true,
		Category:   "grocery",
		Department: "grocery",
		Query:      "fresh",
		Limit:      50,
	})
	if len(filtered) == 0 {
		b.Fatalf("filter returned no deals")
	}
	if err := display.PrintDealsJSON(io.Discard, filtered); err != nil {
		b.Fatalf("print deals json: %v", err)
	}
}

func BenchmarkZipPipeline_1kDeals(b *testing.B) {
	_, client := setupPipelineServer(b, 1000)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		runPipeline(b, client)
	}
}

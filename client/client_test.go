package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ARK21/wert/domain"
)

// --- helpers ---

func newAPIClient(t *testing.T, baseURL string) *APIClient {
	t.Helper()
	cl, err := NewAPIClient(baseURL, "test-key")
	if err != nil {
		t.Fatalf("NewAPIClient: %v", err)
	}
	return cl
}

func mustExchange(t *testing.T, api *APIClient, e domain.Exchange) (float64, error) {
	t.Helper()
	return api.Exchange(context.Background(), e)
}

// --- tests ---

func TestAPIClient_Exchange_OK(t *testing.T) {
	var captured *http.Request

	// Mock CMC API
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r

		// Verify path
		if r.URL.Path != "/v2/tools/price-conversion" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		// Respond with minimal valid payload
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
		  "data": {
		    "USD": {
		      "quote": {
		        "BTC": { "price": 123.45 }
		      }
		    }
		  }
		}`))
	}))
	defer srv.Close()

	api := newAPIClient(t, srv.URL)

	got, err := mustExchange(t, api, domain.Exchange{Amount: 123.45, From: "USD", To: "BTC"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 123.45 {
		t.Fatalf("want 123.45, got %v", got)
	}

	// Check headers set by cmcReq
	if hv := captured.Header.Get("Accept"); hv != "application/json" {
		t.Fatalf("missing/invalid Accept header: %q", hv)
	}
	if hv := captured.Header.Get("X-CMC_PRO_API_KEY"); hv != "test-key" {
		t.Fatalf("missing/invalid API key header: %q", hv)
	}

	// Check query params set by cmcReq
	q := captured.URL.Query()
	if q.Get("amount") != "123.45" { // fmt.Sprintf("%f") prints 6 decimals
		t.Fatalf("amount query mismatch: %q", q.Get("amount"))
	}
	if q.Get("symbol") != "USD" {
		t.Fatalf("symbol query mismatch: %q", q.Get("symbol"))
	}
	if q.Get("convert") != "BTC" {
		t.Fatalf("convert query mismatch: %q", q.Get("convert"))
	}
}

func TestAPIClient_Exchange_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "cmc down", http.StatusInternalServerError)
	}))
	defer srv.Close()

	api := newAPIClient(t, srv.URL)

	_, err := mustExchange(t, api, domain.Exchange{Amount: 1, From: "USD", To: "BTC"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "CMC unexpected status code: 500") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAPIClient_Exchange_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{`)) // invalid JSON
	}))
	defer srv.Close()

	api := newAPIClient(t, srv.URL)

	_, err := mustExchange(t, api, domain.Exchange{Amount: 1, From: "USD", To: "BTC"})
	if err == nil || !strings.Contains(err.Error(), "could not decode data from CMC") {
		t.Fatalf("want decode error, got %v", err)
	}
}

func TestAPIClient_Exchange_MissingDataKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{}}`)) // no "USD"
	}))
	defer srv.Close()

	api := newAPIClient(t, srv.URL)

	_, err := mustExchange(t, api, domain.Exchange{Amount: 1, From: "USD", To: "BTC"})
	if err == nil || !strings.Contains(err.Error(), `missing data for "USD"`) {
		t.Fatalf(`want 'missing data for "USD"', got %v`, err)
	}
}

func TestAPIClient_Exchange_MissingQuoteKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
		  "data": { "USD": { "quote": { } } }
		}`)) // no "BTC"
	}))
	defer srv.Close()

	api := newAPIClient(t, srv.URL)

	_, err := mustExchange(t, api, domain.Exchange{Amount: 1, From: "USD", To: "BTC"})
	if err == nil || !strings.Contains(err.Error(), `missing quote for "BTC"`) {
		t.Fatalf(`want 'missing quote for "BTC"', got %v`, err)
	}
}

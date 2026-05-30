package services

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestESIAssetFetcher_Paginates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Pages", "2")
		switch r.URL.Query().Get("page") {
		case "", "1":
			fmt.Fprint(w, `[{"type_id":34,"location_id":60003760,"quantity":100,"location_flag":"Hangar"}]`)
		case "2":
			fmt.Fprint(w, `[{"type_id":35,"location_id":60003760,"quantity":7,"location_flag":"Hangar"}]`)
		}
	}))
	defer srv.Close()

	f := &ESIAssetFetcher{baseURL: srv.URL, client: srv.Client()}
	got, _, err := f.FetchCharacterAssets(context.Background(), 123, "token")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 2 || got[0].TypeID != 34 || got[0].Quantity != 100 || got[1].TypeID != 35 {
		t.Fatalf("unexpected assets: %+v", got)
	}
}

// TestESIAssetFetcher_ParsesExpiresHeader: ESI signaliert per `Expires`-Header,
// bis wann der serverseitige Cache läuft. Refresh-Klicks vor diesem Zeitpunkt
// liefern dieselben Daten — die UI braucht den Wert, um das anzuzeigen.
func TestESIAssetFetcher_ParsesExpiresHeader(t *testing.T) {
	wantExpiry := time.Date(2026, 5, 30, 15, 0, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Expires", wantExpiry.Format(http.TimeFormat))
		fmt.Fprint(w, `[{"type_id":34,"location_id":60003760,"quantity":1,"location_flag":"Hangar"}]`)
	}))
	defer srv.Close()
	f := &ESIAssetFetcher{baseURL: srv.URL, client: srv.Client()}
	_, gotExpiry, err := f.FetchCharacterAssets(context.Background(), 1, "tok")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !gotExpiry.Equal(wantExpiry) {
		t.Errorf("expiry = %s, want %s", gotExpiry, wantExpiry)
	}
}

func TestESIAssetFetcher_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	f := &ESIAssetFetcher{baseURL: srv.URL, client: srv.Client()}
	if _, _, err := f.FetchCharacterAssets(context.Background(), 1, "bad"); err == nil {
		t.Fatalf("expected error on 401")
	}
}

package apnicstats

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestFetchCountryASNTable_liveTK(t *testing.T) {
	if testing.Short() {
		t.Skip("live fetch")
	}
	if os.Getenv("APNIC_LIVE_TEST") != "1" {
		t.Skip("set APNIC_LIVE_TEST=1 to enable live fetch")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res, err := FetchCountryASNTable(ctx, "TK", http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(res.Rows))
	}
	if res.Rows[0].Name == "" || strings.Contains(res.Rows[0].Name, "</a>") || strings.HasPrefix(res.Rows[0].Name, ",") {
		t.Fatalf("bad name row0: %q", res.Rows[0].Name)
	}
}

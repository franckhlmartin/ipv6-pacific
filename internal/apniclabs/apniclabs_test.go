package apniclabs

import (
	"context"
	"net/http"
	"testing"
)

func TestFetchLatest_invalidISO(t *testing.T) {
	_, err := FetchLatest(context.Background(), "X", http.DefaultClient)
	if err == nil {
		t.Fatal("expected error")
	}
}

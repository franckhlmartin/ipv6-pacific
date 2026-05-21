package ripestat

import (
	"encoding/json"
	"testing"
)

func TestParseData(t *testing.T) {
	data, err := parseData([]byte(`{"status":"ok","data":{"status":"valid"}}`))
	if err != nil {
		t.Fatal(err)
	}
	var row struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &row); err != nil || row.Status != "valid" {
		t.Fatalf("row: %+v err %v", row, err)
	}
	_, err = parseData([]byte(`{"status":"error","message":"nope"}`))
	if err == nil {
		t.Fatal("expected api error")
	}
}

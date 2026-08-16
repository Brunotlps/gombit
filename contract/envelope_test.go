package contract

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestDataJSONShape(t *testing.T) {
	t.Parallel()

	body := Data[string]{Data: "ok"}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"data":"ok"}`
	if string(raw) != want {
		t.Fatalf("json = %s, want %s", raw, want)
	}
	if strings.Contains(string(raw), `"meta"`) {
		t.Fatalf("Data must not emit meta; got %s", raw)
	}
}

func TestDataMetaPageJSONShape(t *testing.T) {
	t.Parallel()

	body := DataMeta[[]string, PageMeta]{
		Data: []string{"a", "b"},
		Meta: &PageMeta{Page: 1, PerPage: 20, Total: 125},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data, ok := got["data"].([]any)
	if !ok || len(data) != 2 || data[0] != "a" || data[1] != "b" {
		t.Fatalf("data = %#v", got["data"])
	}
	meta, ok := got["meta"].(map[string]any)
	if !ok {
		t.Fatalf("meta missing: %#v", got)
	}
	wantMeta := map[string]any{
		"page":     float64(1),
		"per_page": float64(20),
		"total":    float64(125),
	}
	if !reflect.DeepEqual(meta, wantMeta) {
		t.Fatalf("meta = %#v, want %#v", meta, wantMeta)
	}
}

func TestDataMetaOmitsNilMeta(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal(DataMeta[string, PageMeta]{Data: "ok"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(raw) != `{"data":"ok"}` {
		t.Fatalf("json = %s, want data-only when meta is nil", raw)
	}
}

func TestDataMetaZeroPageMetaStillSerializes(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal(DataMeta[string, PageMeta]{
		Data: "ok",
		Meta: &PageMeta{},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"meta"`) {
		t.Fatalf("non-nil zero PageMeta should serialize; got %s", raw)
	}
}

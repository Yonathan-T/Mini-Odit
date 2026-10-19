package router

import (
	"odition/config"
	"testing"
)

func TestRouter(t *testing.T) {
	top := &config.Topology{
		Families: []config.Family{
			{ID: "telebirr", Code: 1, Anchors: []string{"telebirr", "127", "ቴሌብር"}},
			{ID: "boa", Code: 2, Anchors: []string{"bankofabyssinia", "8397"}},
		},
	}
	r := NewRouter(top)

	id, code := r.Route("You transferred 100 ETB via telebirr")
	if id != "telebirr" || code != 1 {
		t.Fatalf("expected telebirr/1, got %s/%d", id, code)
	}

	id, code = r.Route("Receipt: https://cs.bankofabyssinia.com/slip/?trx=123")
	if id != "boa" || code != 2 {
		t.Fatalf("expected boa/2, got %s/%d", id, code)
	}

	id, code = r.Route("Random spam message")
	if id != "" || code != 0 {
		t.Fatalf("expected empty/0 for unknown, got %s/%d", id, code)
	}
}

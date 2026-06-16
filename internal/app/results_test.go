package app

import (
	"encoding/json"
	"testing"
)

const espnSample = `{"events":[{"status":{"type":{"state":"post","completed":true}},
"competitions":[{"competitors":[
{"homeAway":"home","score":"2","team":{"displayName":"Spain","abbreviation":"ESP"}},
{"homeAway":"away","score":"1","team":{"displayName":"Cape Verde","abbreviation":"CPV"}}]}]},
{"status":{"type":{"state":"in","completed":false}},
"competitions":[{"competitors":[
{"homeAway":"home","score":"0","team":{"displayName":"Türkiye","abbreviation":"TUR"}},
{"homeAway":"away","score":"1","team":{"displayName":"Australia","abbreviation":"AUS"}}]}]}]}`

func TestESPNParse(t *testing.T) {
	var r espnResp
	if err := json.Unmarshal([]byte(espnSample), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(r.Events) != 2 {
		t.Fatalf("events = %d, want 2", len(r.Events))
	}
	e0 := r.Events[0]
	if !e0.Status.Type.Completed || e0.Status.Type.State != "post" {
		t.Errorf("event0 status = %+v", e0.Status.Type)
	}
	c := e0.Competitions[0].Competitors
	if c[0].HomeAway != "home" || c[0].Score != "2" || c[0].Team.DisplayName != "Spain" || c[0].Team.Abbreviation != "ESP" {
		t.Errorf("event0 home competitor = %+v", c[0])
	}
	if r.Events[1].Status.Type.State != "in" {
		t.Errorf("event1 should be in-play")
	}
}

func TestResolveESPN(t *testing.T) {
	pair := map[string]int64{"ESP|CPV": 7}
	normToFifa := map[string]string{"spain": "ESP", "capeverde": "CPV"}
	fifaSet := map[string]bool{"ESP": true, "CPV": true}
	// by name
	if id, ok := resolveESPN("Spain", "Cape Verde", "ESP", "CPV", pair, normToFifa, fifaSet); !ok || id != 7 {
		t.Errorf("name resolve = %d,%v", id, ok)
	}
	// by abbreviation fallback (unknown names)
	if id, ok := resolveESPN("???", "???", "ESP", "CPV", pair, normToFifa, fifaSet); !ok || id != 7 {
		t.Errorf("abbrev resolve = %d,%v", id, ok)
	}
	// unmatched
	if _, ok := resolveESPN("Foo", "Bar", "FOO", "BAR", pair, normToFifa, fifaSet); ok {
		t.Errorf("should not match unknown teams")
	}
}

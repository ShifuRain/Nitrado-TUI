package api

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSettings_TolerantDecoding(t *testing.T) {
	var gs GameServer
	body := `{"status":"started","settings":{"general":{"motd":"hi","slots":10},"junk":"x"},"game_specific":{"update_status":"up_to_date"}}`
	if err := json.Unmarshal([]byte(body), &gs); err != nil {
		t.Fatal(err)
	}
	if v, _ := gs.Settings.Get("general", "motd"); v != "hi" {
		t.Errorf("motd = %q", v)
	}
	if v, _ := gs.Settings.Get("general", "slots"); v != "10" {
		t.Errorf("slots = %q, want 10", v)
	}
	if _, ok := gs.Settings["junk"]; ok {
		t.Error("non-object category should be dropped")
	}
	if gs.UpdateStatus() != "up_to_date" {
		t.Errorf("UpdateStatus = %q", gs.UpdateStatus())
	}

	var weird GameServer
	if err := json.Unmarshal([]byte(`{"status":"stopped","settings":[1,2],"game_specific":[]}`), &weird); err != nil {
		t.Fatalf("unexpected shapes must not error: %v", err)
	}
	if weird.Status != "stopped" {
		t.Error("rest of payload should still decode")
	}
}

func TestSettingsEndpoints(t *testing.T) {
	rs := newRecordingServer(t, `{"status":"success","data":{"settings":{"general":{"motd":"x"}}}}`)
	c := New("t", WithBaseURL(rs.URL))

	if err := c.SetSetting(context.Background(), 5, "general", "motd", "hello"); err != nil {
		t.Fatal(err)
	}
	if rs.Method != "POST" || rs.Path != "/services/5/gameservers/settings" ||
		rs.Body["category"] != "general" || rs.Body["key"] != "motd" || rs.Body["value"] != "hello" {
		t.Errorf("SetSetting sent %s %s %v", rs.Method, rs.Path, rs.Body)
	}

	d, err := c.GetSettingsDefaults(context.Background(), 5)
	if err != nil || rs.Path != "/services/5/gameservers/settings/defaults" {
		t.Fatalf("defaults: err=%v path=%s", err, rs.Path)
	}
	if v, _ := d.Get("general", "motd"); v != "x" {
		t.Errorf("default motd = %q", v)
	}

	if err := c.ResetSettings(context.Background(), 5); err != nil || rs.Method != "DELETE" {
		t.Errorf("ResetSettings: err=%v method=%s", err, rs.Method)
	}
}

func TestLifecycleWithMessage(t *testing.T) {
	rs := newRecordingServer(t, `{"status":"success"}`)
	c := New("t", WithBaseURL(rs.URL))
	if err := c.RestartWithMessage(context.Background(), 5, "brb"); err != nil {
		t.Fatal(err)
	}
	if rs.Body["message"] != "brb" || rs.Body["restart_message"] != "brb" {
		t.Errorf("restart body = %v", rs.Body)
	}
	if err := c.StopWithMessage(context.Background(), 5, "bye"); err != nil {
		t.Fatal(err)
	}
	if rs.Path != "/services/5/gameservers/stop" || rs.Body["stop_message"] != "bye" {
		t.Errorf("stop: %s %v", rs.Path, rs.Body)
	}
}

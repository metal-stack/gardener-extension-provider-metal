package bastion

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

func decodeDataURL(t *testing.T, source string) string {
	t.Helper()

	const prefix = "data:,"
	if !strings.HasPrefix(source, prefix) {
		t.Fatalf("unexpected data url %q", source)
	}

	decoded, err := url.PathUnescape(strings.TrimPrefix(source, prefix))
	if err != nil {
		t.Fatalf("failed to decode data url %q: %v", source, err)
	}

	return decoded
}

func Test_transpileUserData(t *testing.T) {
	script := "#!/bin/bash -eu\necho hello bastion\n"

	out, err := transpileUserData([]byte(script))
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(out) {
		t.Fatalf("output is not valid json: %s", out)
	}

	cfg := map[string]any{}
	if err := json.Unmarshal(out, &cfg); err != nil {
		t.Fatalf("output is not valid ignition json: %v", err)
	}

	ignitionSection, ok := cfg["ignition"].(map[string]any)
	if !ok {
		t.Fatalf("expected ignition section, got %v", cfg["ignition"])
	}
	if version := ignitionSection["version"]; version != "2.3.0" {
		t.Errorf("expected ignition version 2.3.0, got %v", version)
	}

	storage, ok := cfg["storage"].(map[string]any)
	if !ok {
		t.Fatalf("expected storage section, got %v", cfg["storage"])
	}
	files, ok := storage["files"].([]any)
	if !ok || len(files) != 3 {
		t.Fatalf("expected 3 files, got %v", storage["files"])
	}

	contents := map[string]string{}
	modes := map[string]float64{}
	overwrite := map[string]bool{}
	for _, f := range files {
		entry, ok := f.(map[string]any)
		if !ok {
			t.Fatalf("unexpected file entry %v", f)
		}
		path, _ := entry["path"].(string)
		fileContents, _ := entry["contents"].(map[string]any)
		source, _ := fileContents["source"].(string)
		contents[path] = decodeDataURL(t, source)
		modes[path], _ = entry["mode"].(float64)
		overwrite[path], _ = entry["overwrite"].(bool)
	}

	if got := contents[hostsAllowPath]; got != "" {
		t.Errorf("expected empty %s, got %q", hostsAllowPath, got)
	}
	if got := contents[hostsDenyPath]; got != "" {
		t.Errorf("expected empty %s, got %q", hostsDenyPath, got)
	}
	if got := contents[userdataScriptPath]; got != script {
		t.Errorf("expected original script at %s, got %q", userdataScriptPath, got)
	}

	if modes[hostsAllowPath] != 0644 || modes[hostsDenyPath] != 0644 {
		t.Errorf("expected hosts files to be 0644, got %v/%v", modes[hostsAllowPath], modes[hostsDenyPath])
	}
	if modes[userdataScriptPath] != 0700 {
		t.Errorf("expected userdata script to be 0700, got %v", modes[userdataScriptPath])
	}

	for _, path := range []string{hostsAllowPath, hostsDenyPath, userdataScriptPath} {
		if !overwrite[path] {
			t.Errorf("expected overwrite=true for %s", path)
		}
	}

	systemd, ok := cfg["systemd"].(map[string]any)
	if !ok {
		t.Fatalf("expected systemd section, got %v", cfg["systemd"])
	}
	units, ok := systemd["units"].([]any)
	if !ok || len(units) != 1 {
		t.Fatalf("expected 1 systemd unit, got %v", systemd["units"])
	}

	unit := units[0].(map[string]any)
	if name := unit["name"]; name != userdataServiceName {
		t.Errorf("expected unit name %q, got %v", userdataServiceName, name)
	}
	if enabled, _ := unit["enabled"].(bool); !enabled {
		t.Error("expected unit to be enabled")
	}
	if got := unit["contents"]; got != userdataServiceContents {
		t.Errorf("unexpected unit contents %q", got)
	}
	if !strings.Contains(userdataServiceContents, "ExecStart="+userdataScriptPath) {
		t.Errorf("expected unit to execute %s, got %q", userdataScriptPath, userdataServiceContents)
	}
}

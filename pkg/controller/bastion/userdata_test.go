package bastion

import (
	"bytes"
	"testing"

	"sigs.k8s.io/yaml"
)

func Test_isCloudConfigFile(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "bash script", in: "#!/bin/bash -eu\necho hi\n", want: false},
		{name: "cloud-config first line", in: "#cloud-config\nruncmd:\n- echo hi\n", want: true},
		{name: "cloud-config second line", in: "#some header\n#cloud-config\nruncmd:\n", want: true},
		{name: "cloud-config too late", in: "line1\nline2\nline3\n#cloud-config\n", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCloudConfigFile([]byte(tt.in)); got != tt.want {
				t.Errorf("isCloudConfigFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ensureEmptyHostsFiles_wrap(t *testing.T) {
	script := "#!/bin/bash -eu\necho hello bastion\n"

	out, err := ensureEmptyHostsFiles([]byte(script))
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.HasPrefix(out, []byte(cloudConfigHeader)) {
		t.Fatalf("expected output to start with %q, got %q", cloudConfigHeader, out)
	}

	cfg := map[string]any{}
	if err := yaml.Unmarshal(out, &cfg); err != nil {
		t.Fatalf("output is not valid cloud-config yaml: %v", err)
	}

	files, ok := cfg["write_files"].([]any)
	if !ok || len(files) != 3 {
		t.Fatalf("expected 3 write_files entries, got %v", cfg["write_files"])
	}

	contents := map[string]string{}
	for _, f := range files {
		entry := f.(map[string]any)
		path, _ := entry["path"].(string)
		content, _ := entry["content"].(string)
		contents[path] = content
	}

	if content, ok := contents[hostsAllowPath]; !ok || content != "" {
		t.Errorf("expected empty content for %s, got %q (present: %v)", hostsAllowPath, content, ok)
	}
	if content, ok := contents[hostsDenyPath]; !ok || content != "" {
		t.Errorf("expected empty content for %s, got %q (present: %v)", hostsDenyPath, content, ok)
	}
	if content, ok := contents[userdataScriptPath]; !ok || content != script {
		t.Errorf("expected original script for %s, got %q", userdataScriptPath, content)
	}

	runcmd, ok := cfg["runcmd"].([]any)
	if !ok || len(runcmd) != 1 || runcmd[0] != userdataScriptPath {
		t.Errorf("unexpected runcmd: %v", cfg["runcmd"])
	}
}

func Test_ensureEmptyHostsFiles_merge(t *testing.T) {
	userData := `#cloud-config
write_files:
- path: /tmp/other
  content: "keep-me"
runcmd:
- echo hi
`

	out, err := ensureEmptyHostsFiles([]byte(userData))
	if err != nil {
		t.Fatal(err)
	}

	cfg := map[string]any{}
	if err := yaml.Unmarshal(out, &cfg); err != nil {
		t.Fatalf("output is not valid cloud-config yaml: %v", err)
	}

	files, ok := cfg["write_files"].([]any)
	if !ok {
		t.Fatalf("expected write_files, got %v", cfg["write_files"])
	}

	contents := map[string]string{}
	for _, f := range files {
		entry := f.(map[string]any)
		path, _ := entry["path"].(string)
		content, _ := entry["content"].(string)
		contents[path] = content
	}

	if contents[hostsAllowPath] != "" || contents[hostsDenyPath] != "" {
		t.Errorf("expected hosts files to be emptied, got %v", contents)
	}
	if contents["/tmp/other"] != "keep-me" {
		t.Errorf("expected /tmp/other to be preserved, got %v", contents)
	}

	runcmd, _ := cfg["runcmd"].([]any)
	if len(runcmd) != 1 || runcmd[0] != "echo hi" {
		t.Errorf("expected original runcmd to be preserved, got %v", cfg["runcmd"])
	}
}

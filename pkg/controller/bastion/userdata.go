package bastion

import (
	"fmt"
	"strings"

	"sigs.k8s.io/yaml"
)

const (
	hostsAllowPath = "/etc/hosts.allow"
	hostsDenyPath  = "/etc/hosts.deny"

	// cloudConfigHeader is the marker that makes the metal os-installer treat the userdata as cloud-config.
	cloudConfigHeader = "#cloud-config"

	// userdataScriptPath is where the original bastion userdata script is stored before it is executed.
	userdataScriptPath = "/var/lib/bastion/userdata.sh"
)

// cloudConfig is a minimal representation of a cloud-init cloud-config. Only the fields relevant for
// the bastion provisioning are modelled, everything else is passed through untouched when merging.
type cloudConfig struct {
	// WriteFiles is the list of files that are written during the provisioning.
	WriteFiles []cloudConfigFile `json:"write_files,omitempty"`
	// RunCmd is the list of commands that are executed during the provisioning.
	RunCmd []string `json:"runcmd,omitempty"`
}

type cloudConfigFile struct {
	Path        string `json:"path"`
	Owner       string `json:"owner,omitempty"`
	Permissions string `json:"permissions,omitempty"`
	Content     string `json:"content"`
}

// ensureEmptyHostsFiles makes sure that the given bastion userdata clears the /etc/hosts.allow and
// /etc/hosts.deny files. These files are pre-populated in the metal-stack OS images (hosts.allow only
// allows internal ranges and hosts.deny denies everything else), which would prevent SSH access to the
// bastion. The original userdata is preserved.
func ensureEmptyHostsFiles(userData []byte) ([]byte, error) {
	if isCloudConfigFile(userData) {
		return mergeHostsFilesIntoCloudConfig(userData)
	}

	return wrapBastionUserData(userData)
}

// isCloudConfigFile returns true if the given userdata should be interpreted as cloud-config by the metal
// os-installer, i.e. if a line containing "#cloud-config" appears within the first two lines.
func isCloudConfigFile(content []byte) bool {
	for i, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, cloudConfigHeader) {
			return true
		}
		if i > 1 {
			return false
		}
	}
	return false
}

// wrapBastionUserData wraps the given (bash) userdata into a cloud-config that writes the empty
// /etc/hosts.allow and /etc/hosts.deny files and afterwards executes the original script.
func wrapBastionUserData(userData []byte) ([]byte, error) {
	cfg := cloudConfig{
		WriteFiles: []cloudConfigFile{
			{
				Path:        hostsAllowPath,
				Owner:       "root:root",
				Permissions: "0644",
				Content:     "",
			},
			{
				Path:        hostsDenyPath,
				Owner:       "root:root",
				Permissions: "0644",
				Content:     "",
			},
			{
				Path:        userdataScriptPath,
				Owner:       "root:root",
				Permissions: "0700",
				Content:     string(userData),
			},
		},
		RunCmd: []string{userdataScriptPath},
	}

	return marshalCloudConfig(cfg)
}

// mergeHostsFilesIntoCloudConfig merges the empty /etc/hosts.allow and /etc/hosts.deny write_files entries
// into an already existing cloud-config userdata.
func mergeHostsFilesIntoCloudConfig(userData []byte) ([]byte, error) {
	cfg := map[string]any{}
	if err := yaml.Unmarshal(userData, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse cloud-config userdata: %w", err)
	}

	var files []any
	if existing, ok := cfg["write_files"].([]any); ok {
		files = existing
	}

	files = upsertEmptyFile(files, hostsAllowPath)
	files = upsertEmptyFile(files, hostsDenyPath)
	cfg["write_files"] = files

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cloud-config userdata: %w", err)
	}

	return append([]byte(cloudConfigHeader+"\n"), out...), nil
}

// upsertEmptyFile ensures that the write_files entry for the given path writes an empty file, preserving
// any other keys of an already existing entry.
func upsertEmptyFile(files []any, path string) []any {
	for _, f := range files {
		entry, ok := f.(map[string]any)
		if !ok {
			continue
		}
		if entry["path"] == path {
			entry["content"] = ""
			return files
		}
	}

	return append(files, map[string]any{
		"path":        path,
		"owner":       "root:root",
		"permissions": "0644",
		"content":     "",
	})
}

func marshalCloudConfig(cfg cloudConfig) ([]byte, error) {
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cloud-config: %w", err)
	}

	return append([]byte(cloudConfigHeader+"\n"), out...), nil
}

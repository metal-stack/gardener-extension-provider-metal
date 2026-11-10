package bastion

import (
	"encoding/json"
	"fmt"

	"github.com/flatcar/container-linux-config-transpiler/config/types"
	"k8s.io/utils/ptr"
)

const (
	hostsAllowPath = "/etc/hosts.allow"
	hostsDenyPath  = "/etc/hosts.deny"

	// userdataScriptPath is where the bastion script is stored before it is executed.
	userdataScriptPath = "/var/lib/bastion/userdata.sh"

	// userdataServiceName is the name of the systemd unit that executes the bastion script.
	userdataServiceName = "bastion-userdata.service"

	// userdataServiceContents is the systemd unit that runs the bastion userdata script.
	userdataServiceContents = `[Unit]
Description=Run metal-stack bastion userdata
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=` + userdataScriptPath + `
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
`
)

// transpileUserData converts the bastion userdata shell script into an ignition configuration, which is
// the provisioning format understood by the metal-stack OS images. Besides running the original script it
// also writes the /etc/hosts.allow and /etc/hosts.deny files as empty files, because the defaults baked
// into the images would otherwise block SSH access to the bastion.
func transpileUserData(userData []byte) ([]byte, error) {
	cfg := types.Config{
		Storage: types.Storage{
			Files: []types.File{
				{
					Filesystem: "root",
					Path:       hostsAllowPath,
					Contents:   types.FileContents{Inline: ""},
					Mode:       ptr.To(0644),
					Overwrite:  ptr.To(true),
				},
				{
					Filesystem: "root",
					Path:       hostsDenyPath,
					Contents:   types.FileContents{Inline: ""},
					Mode:       ptr.To(0644),
					Overwrite:  ptr.To(true),
				},
				{
					Filesystem: "root",
					Path:       userdataScriptPath,
					Contents:   types.FileContents{Inline: string(userData)},
					Mode:       ptr.To(0700),
					Overwrite:  ptr.To(true),
				},
			},
		},
		Systemd: types.Systemd{
			Units: []types.SystemdUnit{
				{
					Name:     userdataServiceName,
					Enabled:  ptr.To(true),
					Contents: userdataServiceContents,
				},
			},
		},
	}

	out, report := types.Convert(cfg, "", nil)
	if report.IsFatal() {
		return nil, fmt.Errorf("could not transpile bastion userdata to ignition: %s", report.String())
	}

	ignition, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("could not marshal ignition config: %w", err)
	}

	return ignition, nil
}

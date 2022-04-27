package main

import (
	"context"
	"fmt"

	"github.com/spiffe/spire-plugin-sdk/pluginmain"
	workloadattestor "github.com/spiffe/spire-plugin-sdk/proto/spire/plugin/agent/workloadattestor/v1"

	"github.com/godbus/dbus/v5"
)

const pluginName = "systemd"

type Plugin struct {
	workloadattestor.UnsafeWorkloadAttestorServer
}

func (p *Plugin) Attest(ctx context.Context, req *workloadattestor.AttestRequest) (*workloadattestor.AttestResponse, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, err
	}

	// Get the unit for the given PID from the systemd service.
	call := conn.Object("org.freedesktop.systemd1", "/org/freedesktop/systemd1").CallWithContext(ctx, "org.freedesktop.systemd1.Manager.GetUnitByPID", 0, uint(req.Pid))
	var unitPath dbus.ObjectPath
	if err := call.Store(&unitPath); err != nil {
		return nil, err
	}

	var selectorValues []string

	// Get the location of the service file
	fragmentPathVariant, err := conn.Object("org.freedesktop.systemd1", unitPath).GetProperty("org.freedesktop.systemd1.Unit.FragmentPath")
	if err != nil {
		return nil, err
	}
	fragmentPath, ok := fragmentPathVariant.Value().(string)
	if !ok {
		return nil, fmt.Errorf("Returned fragment path was not a string: %v", fragmentPathVariant.String())
	}
	selectorValues = append(selectorValues, makeSelectorValue("fragmentPath", fragmentPath))

	// TODO(zecke): Add other interesting bits of the unit.

	return &workloadattestor.AttestResponse{
		SelectorValues: selectorValues,
	}, nil
}

func makeSelectorValue(kind, value string) string {
	return fmt.Sprintf("%s:%s", kind, value)
}

func main() {
	plugin := new(Plugin)
	// Serve the plugin. This function call will not return. If there is a
	// failure to serve, the process will exit with a non-zero exit code.
	pluginmain.Serve(
		workloadattestor.WorkloadAttestorPluginServer(plugin),
	)
}

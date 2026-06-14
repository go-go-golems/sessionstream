package provider

import (
	"encoding/json"
	"fmt"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	ssmodule "github.com/go-go-golems/sessionstream/pkg/js/modules/sessionstream"
	ss "github.com/go-go-golems/sessionstream/pkg/sessionstream"
)

const (
	PackageID      = "sessionstream"
	HostServiceKey = "sessionstream.host-options.v1"
)

type HostOptions struct {
	HubOptions []ss.HubOption
}

var configSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "websocket": {
      "type": "object",
      "properties": {
        "enabled": {"type": "boolean"},
        "path": {"type": "string"},
        "maxHydrationBufferedBatches": {"type": "integer", "minimum": 1}
      },
      "additionalProperties": false
    },
    "schemas": {"type": "object"}
  },
  "additionalProperties": false
}`)

func Register(registry *providerapi.ProviderRegistry) error {
	return registry.Package(PackageID, providerapi.Module{
		Name:         ssmodule.ModuleName,
		DefaultAs:    ssmodule.ModuleName,
		Description:  "sessionstream Hub, schemas, projections, fanout, and WebSocket helpers exposed as require(\"sessionstream\").",
		ConfigSchema: configSchema,
		TypeScript:   ssmodule.TypeScriptModule(),
		NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
			hostOpts, err := hostOptionsFromServices(ctx.Host)
			if err != nil {
				return nil, err
			}
			return ssmodule.NewLoader(ssmodule.Options{RuntimeOwner: ctx.RuntimeOwner, DefaultHubOptions: hostOpts.HubOptions}), nil
		},
	})
}

func hostOptionsFromServices(host providerapi.HostServices) (HostOptions, error) {
	lookup, ok := host.(providerapi.HostServiceLookup)
	if !ok || lookup == nil {
		return HostOptions{}, nil
	}
	raw, ok := lookup.HostService(HostServiceKey)
	if !ok || raw == nil {
		return HostOptions{}, nil
	}
	opts, ok := raw.(HostOptions)
	if !ok {
		return HostOptions{}, fmt.Errorf("sessionstream host service %q must be provider.HostOptions, got %T", HostServiceKey, raw)
	}
	return opts, nil
}

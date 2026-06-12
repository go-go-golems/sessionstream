package provider

import (
	"encoding/json"

	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	ssmodule "github.com/go-go-golems/sessionstream/pkg/js/modules/sessionstream"
)

const PackageID = "sessionstream"

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
			return ssmodule.NewLoader(ssmodule.Options{RuntimeOwner: ctx.RuntimeOwner}), nil
		},
	})
}

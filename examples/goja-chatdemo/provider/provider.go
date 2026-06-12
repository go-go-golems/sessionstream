package provider

import (
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	chatdemov1 "github.com/go-go-golems/sessionstream/examples/chatdemo/gen/sessionstream/examples/chatdemo/v1"
)

const (
	PackageID  = "sessionstream-chatdemo"
	ModuleName = "sessionstream.examples.chatdemo.v1"
)

// Register exposes the generated chatdemo protobuf builders as an xgoja provider
// module. The provider is intentionally small: it proves sessionstream schemas can
// be authored from JavaScript with the shared go-go-goja protobuf builder runtime
// before the broader require("sessionstream") module is implemented.
func Register(registry *providerapi.ProviderRegistry) error {
	return registry.Package(PackageID, providerapi.Module{
		Name:        ModuleName,
		Description: "Generated Goja protobuf builders for the sessionstream chatdemo schema",
		TypeScript:  chatdemov1.GojaBuilderFileChatProtoTypeScriptModule(ModuleName),
		NewModuleFactory: func(providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
			return chatdemov1.NewGojaBuilderFileChatProtoLoader(ModuleName), nil
		},
	})
}

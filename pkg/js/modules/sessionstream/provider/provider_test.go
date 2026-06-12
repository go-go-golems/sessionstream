package provider

import (
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/tsgen/render"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/dtsgen"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	ssmodule "github.com/go-go-golems/sessionstream/pkg/js/modules/sessionstream"
	"github.com/stretchr/testify/require"
)

func TestProviderRegistersSessionstreamModule(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	require.NoError(t, Register(registry))
	module, ok := registry.ResolveModule(PackageID, ssmodule.ModuleName)
	require.True(t, ok)
	require.NotNil(t, module.TypeScript)
	require.NotEmpty(t, module.ConfigSchema)
	dts, err := render.Bundle(&spec.Bundle{Modules: []*spec.Module{module.TypeScript}})
	require.NoError(t, err)
	require.Contains(t, dts, "declare module \"sessionstream\"")
	require.Contains(t, dts, "interface Hub")
	bundle, err := dtsgen.RenderModules(registry, []dtsgen.ModuleInstance{{Package: PackageID, Name: ssmodule.ModuleName}}, dtsgen.Options{})
	require.NoError(t, err)
	require.Contains(t, bundle.DTS, "eventEmitterFanout")
	loader, err := module.NewModuleFactory(providerapi.ModuleSetupContext{Name: ssmodule.ModuleName})
	require.NoError(t, err)
	require.NotNil(t, loader)
}

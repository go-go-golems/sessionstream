package provider

import (
	"strings"
	"testing"

	"github.com/dop251/goja"
	noderequire "github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/protogoja"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/render"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/dtsgen"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	chatdemov1 "github.com/go-go-golems/sessionstream/examples/chatdemo/gen/sessionstream/examples/chatdemo/v1"
	"github.com/stretchr/testify/require"
)

func TestProviderRegistersChatdemoProtobufBuilderModule(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	require.NoError(t, Register(registry))

	module, ok := registry.ResolveModule(PackageID, ModuleName)
	require.True(t, ok, "provider module %s/%s should be registered", PackageID, ModuleName)
	require.NotNil(t, module.TypeScript)

	dts, err := render.Bundle(&spec.Bundle{Modules: []*spec.Module{module.TypeScript}})
	require.NoError(t, err)
	for _, want := range []string{
		"declare module \"sessionstream.examples.chatdemo.v1\"",
		"export interface StartInferenceCommandBuilder",
		"prompt(value: string): this;",
		"export const ChatMessageEntity",
	} {
		require.Contains(t, dts, want)
	}

	result, err := dtsgen.RenderModules(registry, []dtsgen.ModuleInstance{{Package: PackageID, Name: ModuleName}}, dtsgen.Options{})
	require.NoError(t, err)
	require.Contains(t, result.DTS, "declare module \"sessionstream.examples.chatdemo.v1\"")

	loader, err := module.NewModuleFactory(providerapi.ModuleSetupContext{Name: ModuleName})
	require.NoError(t, err)
	vm := goja.New()
	requireRegistry := noderequire.NewRegistry()
	requireRegistry.RegisterNativeModule(ModuleName, loader)
	requireRegistry.Enable(vm)

	value, err := vm.RunString(`
		const pb = require("sessionstream.examples.chatdemo.v1");
		pb.StartInferenceCommand.builder().prompt("from provider").build();
	`)
	require.NoError(t, err)
	msg, ok := protogoja.MessageFromValue(value)
	require.True(t, ok, "provider-built value should carry a proto ref")
	cmd, ok := msg.(*chatdemov1.StartInferenceCommand)
	require.True(t, ok, "message type = %T", msg)
	require.Equal(t, "from provider", cmd.GetPrompt())

	require.True(t, strings.Contains(result.DTS, "StartInferenceCommandBuilder"))
}

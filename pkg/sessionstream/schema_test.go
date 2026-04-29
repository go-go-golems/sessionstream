package sessionstream

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestSchemaRegistryRejectsDuplicateRegistration(t *testing.T) {
	r := NewSchemaRegistry()
	require.NoError(t, r.RegisterCommand("LabStart", &structpb.Struct{}))
	require.Error(t, r.RegisterCommand("LabStart", &structpb.Struct{}))
}

func TestSchemaRegistryDecodeCommandJSON(t *testing.T) {
	r := NewSchemaRegistry()
	require.NoError(t, r.RegisterCommand("LabStart", &structpb.Struct{}))
	msg, err := r.DecodeCommandJSON("LabStart", []byte(`{"prompt":"hello"}`))
	require.NoError(t, err)
	payload := msg.(*structpb.Struct).AsMap()
	require.Equal(t, "hello", payload["prompt"])
}

func TestSchemaRegistryClonesRegisteredAndReturnedPrototypes(t *testing.T) {
	r := NewSchemaRegistry()
	prototype, err := structpb.NewStruct(map[string]any{"prompt": "original"})
	require.NoError(t, err)
	require.NoError(t, r.RegisterCommand("LabStart", prototype))

	prototype.Fields["prompt"] = structpb.NewStringValue("mutated-after-register")

	lookedUp, ok := r.CommandSchema("LabStart")
	require.True(t, ok)
	require.Equal(t, "original", lookedUp.(*structpb.Struct).AsMap()["prompt"])
	lookedUp.(*structpb.Struct).Fields["prompt"] = structpb.NewStringValue("mutated-after-lookup")

	lookedUpAgain, ok := r.CommandSchema("LabStart")
	require.True(t, ok)
	require.Equal(t, "original", lookedUpAgain.(*structpb.Struct).AsMap()["prompt"])
}

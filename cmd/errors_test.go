package cmd_test

import (
	"testing"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/builder"
	"github.com/strangelove-ventures/cosmos-client/cmd"
	"github.com/stretchr/testify/require"
)

func TestGRPCServiceNotFoundError(t *testing.T) {
	e := cmd.GRPCServiceNotFoundError{
		Requested: "svc1",
		Available: []string{"svc2", "svc3"},
	}

	require.Equal(
		t,
		`no service "svc1" found (available services: svc2, svc3)`,
		e.Error(),
	)
}

func TestGRPCMethodNotFoundError(t *testing.T) {
	// Need to use some dynamic descriptor generation
	// to satisfy the error's Available field.

	svc := builder.NewService("farm")

	b := builder.NewMethod(
		"Moo",
		builder.RpcTypeMessage(builder.NewMessage("MooRequest"), false),
		builder.RpcTypeMessage(builder.NewMessage("MooResponse"), false),
	)
	svc.AddMethod(b)
	moo, err := b.Build()
	require.NoError(t, err)

	b = builder.NewMethod(
		"Baa",
		builder.RpcTypeMessage(builder.NewMessage("BaaRequest"), false),
		builder.RpcTypeMessage(builder.NewMessage("BaaResponse"), false),
	)
	svc.AddMethod(b)
	baa, err := b.Build()
	require.NoError(t, err)

	e := cmd.GRPCMethodNotFoundError{
		TargetService: "farm",
		Requested:     "Ribbit",
		Available:     []*desc.MethodDescriptor{moo, baa},
	}

	require.Equal(
		t,
		`service "farm" has no method with name "Ribbit" (available methods: Baa, Moo)`,
		e.Error(),
	)
}

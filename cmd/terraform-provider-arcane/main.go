package main

import (
	"context"
	"flag"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-arcane/internal/provider"
)

// Set by goreleaser or -ldflags
var (
	version = "dev"
)

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	ctx := context.Background()
	if v := os.Getenv("TF_LOG"); v != "" {
		tflog.Info(ctx, "Starting arcane provider", map[string]any{"version": version})
	}

	providerserver.Serve(ctx, provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/hellscrimson/arcane",
		Debug:   debug,
	})
}

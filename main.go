package main

import (
	"context"
	"flag"
	"log"

	"github.com/articulate/terraform-provider-litellm/internal/metadata"
	"github.com/articulate/terraform-provider-litellm/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// version is set during the release process via -ldflags
var version string = "dev"

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), serveOptions(debug))
	if err != nil {
		log.Fatal(err.Error())
	}
}

func serveOptions(debug bool) providerserver.ServeOpts {
	return providerserver.ServeOpts{
		Address: metadata.ProviderSource,
		Debug:   debug,
	}
}

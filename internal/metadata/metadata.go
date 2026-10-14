// Package metadata contains release and compatibility metadata shared by the
// provider entrypoint, documentation contracts, and release tooling.
//
// It is internal implementation detail, not a supported Go API.
package metadata

const (
	// ProviderSource is the exact source published in the provider registry.
	ProviderSource = "terraform.articulate.team/articulate/litellm"

	// ProtocolVersion is the Terraform provider protocol advertised by releases.
	ProtocolVersion = "6.0"

	// Baseline client and development versions.
	TerraformMinimum = "1.1.0"
	GoMinimum        = "1.24.0"

	// WriteOnlyClientMinimum applies only when optional write-only attributes are
	// configured. It is deliberately not the global client minimum.
	WriteOnlyClientMinimum = "1.11.0"

	// TestedLiteLLMVersion is the exact backend release used by the acceptance
	// harness. Other backend versions are not implied by this literal.
	TestedLiteLLMVersion = "1.104.0"

	// CurrentProviderVersion is the release prepared by this source tree.
	CurrentProviderVersion = "3.0.0"

	// MinimumProviderVersion is the stable lower bound for published examples.
	// It must not move merely because a later compatible 3.x release is tagged.
	MinimumProviderVersion    = "3.0.0"
	ExampleProviderConstraint = ">= 3.0.0, < 4.0.0"
	TerraformRequiredVersion  = ">= 1.1.0"
)

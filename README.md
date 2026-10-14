<p align="center">
  <img src="assets/logo.png" alt="LiteLLM Terraform provider logo" width="144" height="144">
</p>

<h1 align="center">LiteLLM Terraform provider</h1>

<p align="center">Manage your LiteLLM gateway with Terraform or OpenTofu.</p>

<p align="center">
  <a href="https://github.com/articulate/terraform-provider-litellm/actions/workflows/ci.yml"><img src="https://github.com/articulate/terraform-provider-litellm/actions/workflows/ci.yml/badge.svg?branch=main" alt="CI status"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-blue" alt="Apache 2.0 license"></a>
</p>

Articulate's fork of [ncecere/terraform-provider-litellm](https://github.com/ncecere/terraform-provider-litellm)
manages models, teams, keys, budgets, and other gateway resources through the
LiteLLM REST API. Keep model routing and access controls in reviewed configuration,
inspect changes in a plan, and import existing resources.

The first fork version, **3.0.0**, contains upstream v2.2.0 provider behavior.
Registry publication is not enabled yet. Until publication, [build the provider
locally](#development). The source address is
`terraform.articulate.team/articulate/litellm`.

[Quick start](#quick-start) · [Migration](#migrate-from-upstream) ·
[Resource reference](#resource-reference) · [Security](#security) ·
[Development](#development)

## Quick start

Use an existing LiteLLM proxy and a key with permission to read its model
configuration. Supply the proxy URL and key through the `LITELLM_API_BASE` and
`LITELLM_API_KEY` environment variables. Obtain the key from your secret manager.

Save this configuration as `main.tf`:

```hcl
terraform {
  required_version = ">= 1.1.0"

  required_providers {
    litellm = {
      source  = "terraform.articulate.team/articulate/litellm"
      version = ">= 3.0.0, < 4.0.0"
    }
  }
}

provider "litellm" {}

data "litellm_models" "available" {}

output "model_count" {
  value = length(data.litellm_models.available.models)
}
```

After installing the provider locally, or once a registry release is available,
initialize and plan:

```sh
terraform init
terraform plan
```

The plan reports `model_count`, the number of model configurations visible to
your key. This example reads the proxy without creating or changing its resources.
OpenTofu users can run `tofu init` and `tofu plan` instead.

For an existing upstream configuration, complete the [state migration](#migrate-from-upstream)
before the first plan. After changing a version constraint, run
`terraform init -upgrade` or `tofu init -upgrade`.

### Manage a team and key

Add these resources to grant a team access to an existing model alias. Replace
`existing-model-alias` with a model name configured on your proxy.

```hcl
resource "litellm_team" "application" {
  team_alias      = "application"
  models          = ["existing-model-alias"]
  max_budget      = 100.0
  budget_duration = "30d"
}

resource "litellm_key" "application" {
  key_alias       = "application"
  team_id         = litellm_team.application.team_id
  max_budget      = 25.0
  budget_duration = "30d"
}
```

When `team_id` is set and `models` is omitted, the provider sends
`["all-team-models"]` so the key can use the team's models. The generated key
is a sensitive value in state. See the [team](docs/resources/team.md) and
[key](docs/resources/key.md) references for rate limits, route permissions,
service accounts, and write-only keys.

## Compatibility

- [Terraform](https://developer.hashicorp.com/terraform/install) >= 1.1.0 (provider protocol 6.0)
- [OpenTofu](https://opentofu.org/docs/intro/install/) >= 1.6.0
- [Go](https://go.dev/doc/install) >= 1.24.0 for provider development
- Tested backend: exactly LiteLLM 1.104.0

Terraform 1.0 is unsupported. Its `ignore_changes` planning behavior can erase
the distinction between an omitted map and an explicitly empty map before
provider dispatch.

The optional write-only attributes `litellm_key.key_wo`,
`litellm_key.send_invite_email`, `litellm_user.send_invite_email`, and
`litellm_jwt_key_mapping.key_wo` require Terraform or OpenTofu 1.11.0 or later
only when configured. They do not raise the global client minimum.

LiteLLM organization endpoints, including reads, require an Enterprise license
from LiteLLM 1.102.0. Review the [upgrade notes](CHANGELOG.md) before changing
the provider or proxy version.

## Migrate from upstream

The fork keeps the resource types, state format, IDs, and import formats of
upstream v2.2.0. The provider source address changes. Every existing state must
replace its upstream address before the first plan with the fork.

For Terraform:

```sh
terraform state replace-provider \
  registry.terraform.io/ncecere/litellm \
  terraform.articulate.team/articulate/litellm
```

For OpenTofu state created with the short `ncecere/litellm` source:

```sh
tofu state replace-provider \
  registry.opentofu.org/ncecere/litellm \
  terraform.articulate.team/articulate/litellm
```

If OpenTofu used an explicit Terraform Registry source, use
`registry.terraform.io/ncecere/litellm` as the source argument. Match the address
recorded in state. Both commands use the same target and change the provider
association without recreating gateway resources.

Update the source and version constraint in configuration, then initialize
and inspect the plan. The [3.0.0 upgrade notes](CHANGELOG.md#300) describe the
migration and compatibility baseline.

## Resource reference

Use the resource documentation for complete arguments, import formats, and
lifecycle limitations. Single-item and list lookups are in the
[data-source reference](docs/index.md#available-data-sources).

| Area | Resources |
| --- | --- |
| Models and routing | [Models](docs/resources/model.md), [fallbacks](docs/resources/fallback.md), [model access groups](docs/resources/access_group.md), [unified access groups](docs/resources/unified_access_group.md) |
| Identity and access | [Keys](docs/resources/key.md), [key blocking](docs/resources/key_block.md), [users](docs/resources/user.md), [teams](docs/resources/team.md), [team blocking](docs/resources/team_block.md), [team members](docs/resources/team_member.md), [batch members](docs/resources/team_member_add.md), [JWT mappings](docs/resources/jwt_key_mapping.md) |
| Budgets and organization | [Budgets](docs/resources/budget.md), [organizations](docs/resources/organization.md), [organization members](docs/resources/organization_member.md), [projects](docs/resources/project.md), [tags](docs/resources/tag.md) |
| Tools and knowledge | [MCP servers](docs/resources/mcp_server.md), [agents](docs/resources/agent.md), [search tools](docs/resources/search_tool.md), [vector stores](docs/resources/vector_store.md) |
| Provider configuration | [Credentials](docs/resources/credential.md), [guardrails](docs/resources/guardrail.md), [prompts](docs/resources/prompt.md) |

Model configuration includes modes, reasoning effort, and Bedrock cross-account
access through `aws_session_name` and `aws_role_name`. Follow the
[model reference](docs/resources/model.md) for provider-specific settings.
The [examples directory](examples/README.md) contains complete configurations.

## Security

Use environment variables or a secret manager for provider credentials. The
provider also accepts `api_base` and sensitive `api_key` arguments in HCL.
Explicit arguments take precedence over environment variables. TLS certificate
verification is enabled by default. `insecure_skip_verify` disables it.

Terraform's `sensitive` flag hides values in normal output. It does not keep
them out of state. Protect state and plan files with encryption and access
controls. Generated keys and configured credentials can persist there.

For predefined keys, `key_wo` can avoid storing plaintext in the key resource.
Use an ephemeral source so the input does not persist elsewhere in the plan or
state. See [write-only key handling](docs/resources/key.md#write-only-predefined-key)
for client requirements, key entropy, and replacement behavior.

Do not commit API keys, Terraform state, local variable files, or private logs.
Report vulnerabilities through the [Articulate security policy](https://github.com/articulate/.github/blob/main/SECURITY.md).

## Development

Build and install the provider from source:

```sh
git clone https://github.com/articulate/terraform-provider-litellm.git
cd terraform-provider-litellm
go mod download
make install
```

`make install` puts the binary in the local plugin directory for
`terraform.articulate.team/articulate/litellm`, version `3.0.0`.
The repository distributes a provider executable and HCL interface, not a
supported Go library API.

Run the local checks:

```sh
make test
make vet
make contract-check
```

CI also checks formatting, runs race tests, and builds release snapshots.
Provider code lives in `internal/provider`. `main.go` registers the Terraform
Plugin Framework implementation. Resource and data-source references live in
`docs/`, and examples live in `examples/`.

The checked OpenAPI contract and offline verifier protect the provider's HTTP
API boundary without a runtime Python or upstream-source dependency. Read
[API contract maintenance](docs/development/api-contract.md) before updating
the pinned LiteLLM contract. `make contract-diff` compares a reproduction.
`make contract-update` regenerates the checked artifacts after review.
`make contract-update-atomicity-test` checks locking and rollback behavior.

Smoke and acceptance tests create and destroy real resources. Use the pinned
disposable LiteLLM stack and the explicit safety confirmations in
[the test harness guide](internal_testing/README.md). Private smoke logs are
written under `internal_testing/.smoke-logs/`.

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a change. Release history
and compatibility changes are in [CHANGELOG.md](CHANGELOG.md).

## License

The provider uses [Apache License 2.0](LICENSE).
[NOTICE](NOTICE) credits the upstream project.

# Release the provider

The [release workflow](../../.github/workflows/release.yml) publishes
`terraform.articulate.team/articulate/litellm` from a reviewed `v*` tag on the
default branch. It runs the exact-tag acceptance matrix with Terraform 1.1.0
and 1.11.4 against disposable LiteLLM 1.104.0 before signing or publication.

GoReleaser builds the archives, registry manifest, checksums, and detached GPG
signature. The workflow stages every checksum entry beside the checksum file
and checks its contents before AWS authentication. boring-registry 0.16.6 then
checks those files and the signature against the namespace's registered keys.
The GitHub release stays a draft until the registry upload succeeds.

## Configure publication

Obtain approval before creating a signing key, changing registry signing keys,
or publishing a version. These steps do not belong in a pull-request job.

1. Create a dedicated LiteLLM release-signing key. Keep its backup in the team's
   secret manager. Do not reuse the upstream or Kong private key.
2. Create the `provider-release` GitHub environment. Require a Platform reviewer
   and restrict deployment to reviewed `v*` tags. Protect those tags from
   unauthorized creation, updates, and deletion through repository rulesets.
3. Configure these values:

   | Setting | Scope | Value |
   | --- | --- | --- |
   | `GPG_PRIVATE_KEY` | Environment secret | ASCII-armored dedicated private key |
   | `PASSPHRASE` | Environment secret | Passphrase for that private key |
   | `PROVIDER_GPG_FINGERPRINT` | Repository variable | Full 40-character public fingerprint |

4. Read `providers/articulate/signing-keys.json` from the registry bucket through
   approved AWS tooling. Add the new public key to `gpg_public_keys`. Preserve
   every existing key, including Kong's key. Each entry contains `key_id` and
   `ascii_armor`. Verify the new key's full fingerprint before the approved write.

The registry bucket is `tf-registry-us-east-1-233118049495` in `us-east-1`.
The workflow reuses the existing `github-actions` OIDC role in that account.
Its session policy permits bucket listing, namespace public-key reads, and
LiteLLM artifact reads and uploads. It cannot write signing keys, Kong files,
modules, or unrelated objects. It does not grant object deletion.

The role's trust policy uses `repo:articulate@766431/*:*`, which covers this
repository's immutable OIDC subject. Confirm the deployed trust and the
repository's OIDC subject settings before a release. A local source check does
not prove live AWS configuration. No new AWS access key is required.

The upstream signing pins in `internal_testing/upgrade_matrix/tools.lock.json`
authenticate the previous provider used by upgrade tests. Keep those pins
separate from Articulate's release signer.

See boring-registry's [provider publishing guide](https://github.com/boring-registry/boring-registry/blob/v0.16.6/docs/tasks/publish-providers.md)
for the namespace key format. S3 requires bucket-list permission to distinguish
an absent upload destination from an inaccessible object in a
[HeadObject request](https://docs.aws.amazon.com/AmazonS3/latest/API/API_HeadObject.html).

## Publish a version

Release Please maintains the next version and changelog in a release PR after
conventional commits reach `main`. Just before merging that PR, add non-empty
`Upgrade Notes` and `Security` subsections to its changelog entry. CI checks
the manifest's exact version with the same extractor that publication uses.
New commits on `main` can regenerate the release PR and overwrite those edits.
Merging creates a draft GitHub release and its tag. The existing tag workflow
runs acceptance and signing, replaces the draft body with the reviewed changelog
entry, then publishes only after registry upload succeeds.

The automation starts after the manual `3.1.0` release. Before merging its
configuration, grant this repository access to `RUNNER_SMALL`,
`AUTOMATION_CLIENT_ID`, and `AUTOMATION_PRIVATE_KEY`. The App token needs contents
and pull-request write permissions so its release PRs and tags trigger CI.
Allow that App in the `Release tag creation` ruleset. Keep release tags immutable.

For a manual release:

1. Prepare the version's `CHANGELOG.md` section in a reviewed PR. Include
   non-empty `Upgrade Notes` and `Security` subsections. The first fork release
   uses the existing `3.0.1` section.
2. After CI passes on the chosen main commit, create and push its version tag.
   For the first release, use `v3.0.1`. The unpublished `v3.0.0` tag failed its
   release gate. Never move an existing release tag.
3. Inspect the exact-tag matrix, then approve the `provider-release` environment
   deployment. The workflow checks that the imported key matches the configured
   fingerprint before GoReleaser signs anything.
4. Confirm that the registry upload and GitHub publication succeed. Initialize
   a disposable Terraform configuration through `art` with the exact source and
   version. Confirm that Terraform reports the expected signing key. Initialize
   the Kong provider too, to check that its namespace key remains valid.

Only a successful signed installation proves distribution. A snapshot build
or a GitHub release alone does not. Do not migrate consumer states until the
installation checks pass. Record the release commit, version, full fingerprint,
and installation results in the delivery receipt.

## Recover a failed release

The workflow serializes releases and boring-registry refuses existing objects.
Do not run another publisher concurrently. The uploader checks existence before
writing but does not provide an atomic transaction across all release files.

Inspect the failed job before retrying. The workflow never deletes or replaces
release assets or registry objects automatically.

- Before registry upload, a failure can leave a GitHub draft with assets.
  Reusing that draft does not authorize replacing assets. After confirming the
  registry has no files for the version, obtain approval to remove the stale
  draft before rebuilding the same tag.
- During upload, a failure can leave part of the version in the registry.
  Keep the GitHub release a draft. Inspect the exact version's objects before
  choosing a new version or an approved cleanup. Do not rerun blindly.
- After a successful upload, a GitHub publication failure needs no rebuild.
  Verify the uploaded version's checksums and signature, then publish the
  existing draft with approval. Do not upload the version again.
- For signer or OIDC errors, correct the configuration before any retry.
  Do not change the upstream upgrade-test pins to bypass a signing failure.

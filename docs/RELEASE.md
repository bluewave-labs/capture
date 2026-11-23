# Release process for Capture

Releases are cut from `develop`, merged into `main`, and published from a version
tag `vX.Y.Z` (for example `v2.0.0`). Versions follow [semantic versioning](https://semver.org):
a change to an existing API response shape is a major version.

## What publishes what

| Event | What happens |
| ----- | ------------ |
| Tag `vX.Y.Z` pushed | `release.yml` runs the tests, then GoReleaser builds the binaries, SBOMs and checksums into a **draft** GitHub release and pushes the image `ghcr.io/bluewave-labs/capture:vX.Y.Z` |
| Draft release published | The install scripts (`deployment/*/install.sh`, `install.ps1`) pick it up, since they download the latest published release. `release-latest.yml` points the image's `latest` tag at `vX.Y.Z`, which Docker users and the Helm chart (default tag `latest`) follow |

Nothing reaches users through `latest` until the release is published. Still,
push the tag only when the release notes are final, and publish soon after.

## Steps

1. **Check develop is ready.** Everything meant for the release is merged and CI on
   `develop` is green. Locally, with the Go version from `go.mod`:

    ```shell
    go build ./... && go vet ./...
    go test -count=1 -race ./...
    go run golang.org/x/vuln/cmd/govulncheck@v1.10.0 -format json ./... > vuln.json
    .github/scripts/govulncheck-gate.sh vuln.json
    ```

2. **Write the release notes** in the format below. Start from GitHub's generated
   notes, which give the merged PR list, new contributors and the compare link:

    ```shell
    gh api -X POST repos/bluewave-labs/capture/releases/generate-notes \
      -f tag_name=vX.Y.Z -f target_commitish=develop -f previous_tag_name=vPREV \
      --jq .body
    ```

3. **Update `CHANGELOG.md`** with the same notes under `## X.Y.Z (YYYY-MM-DD)`,
   newest first. Open a PR from `docs/changelog-vX.Y.Z` into `develop` and merge it.

4. **Merge `develop` into `main`.** Open a PR titled `Develop -> Main | vX.Y.Z Release`
   and merge it with **Create a merge commit**. Do not squash or rebase it, or `main`
   and `develop` diverge.

5. **Tag the release** on the `develop` commit that was merged into `main` (the
   changelog merge):

    ```shell
    git fetch origin
    git tag vX.Y.Z origin/develop
    git push origin vX.Y.Z
    ```

6. **Check the draft** once the `release` workflow finishes: 17 assets (8 archives,
   8 SBOMs, `checksums.txt`), and `capture --version` from one archive shows the new
   version. Paste the release notes into the draft.

7. **Publish the draft.** Then confirm the `release-latest` workflow ran and that
   `docker buildx imagetools inspect ghcr.io/bluewave-labs/capture:latest` shows the
   same digest as `:vX.Y.Z`.

8. **Tell Checkmate.** If the Checkmate docs or UI mention a Capture version, open
   an issue or PR there.

## Release notes format

Write for the people running Capture: what they get, what changed for them, and
what they have to do. Use the same layout as the Checkmate release notes.

```markdown
## What's Changed

**<The one to three changes that matter most>.** <One or two sentences on what
this means for users and which Checkmate version it pairs with, if any.>

  ### Breaking changes

  - <What changed, what breaks and what to do. Name the endpoint and field>

  ### Features

  - <What users can now do or see>

  ### Fixes

  - <The problem users saw, and what happens now>

  ### Upgrade notes

  - <New or changed environment variables, defaults, install-script or service
    changes. Say "No new environment variables" when there are none>

  ### Maintenance

  - <Dependencies, Go version, CI, tests and docs, grouped>

## New Contributors
* @login made their first contribution in https://github.com/bluewave-labs/capture/pull/N

**Full Changelog**: https://github.com/bluewave-labs/capture/compare/vPREV...vX.Y.Z
```

- No commit hashes, PR numbers or commit prefixes (`feat:`, `fix:`) in the bullets.
- One change per bullet, sentence case, no trailing period.
- Leave out empty sections, except "Upgrade notes". "Breaking changes" comes first
  when there is one, and the opening sentence mentions it.
- Say which platform a change applies to when it is not all of them ("Linux only",
  "Windows service").

## Troubleshooting

- **The `release` workflow fails before GoReleaser runs** (for example in tests):
  nothing was published. Fix it on `develop`, then delete and push the tag again:

    ```shell
    git tag -d vX.Y.Z
    git push --delete origin vX.Y.Z
    ```

- **It fails after the image was pushed:** prefer releasing a new patch version
  over re-tagging, so one version never names two different builds.
- **GoReleaser fails because the release already exists:** delete the draft
  release, then re-run the workflow.
- **`latest` did not move:** run the `release-latest` workflow by hand with the
  release tag. It only moves `latest` to the newest published release.

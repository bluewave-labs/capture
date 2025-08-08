# Release Process for Capture

This document outlines the steps to create a new release for the Capture.

You must follow these steps to ensure a smooth release process.

- Tag Pattern: `v{0-9}.{0-9}.{0-9}`
  - Example: `v1.3.0`

1. Create a new tag for the release version.

    ```shell
    git tag <version>
    ```

2. Push the tag to the remote repository.

    ```shell
    git push origin <version>
    ```

3. [Create a PR](https://github.com/bluewave-labs/capture/compare/main...develop) for merging `develop` into `main`. Merge it with 'Merge Commit' option, do not squash or rebase.

4. Update the `CHANGELOG.md` file with the new version details.

5. Commit the changes to the `CHANGELOG.md` file.

    ```shell
    git switch -c docs/changelog-<version>
    git add CHANGELOG.md
    git commit -m "docs(changelog): Update CHANGELOG for version <version>"
    git push origin docs/changelog-<version>
    ```

6. Create a PR for the `docs/changelog-<version>` branch to merge into `develop`.

7. Close the milestone for the current version.

## Troubleshooting

If you encounter issues during the release process, consider following these steps:

1. Check the GitHub Actions logs for any errors or warnings.
2. Ensure that all required environment variables are set correctly.
3. Verify that the version tag follows the correct format (e.g., `v1.0.0`).
4. Try to re-run the release workflow if it fails due to transient issues.
5. If the release process fails, you can revert the changes by deleting the tag and branch created during the release.

    ```shell
    git tag -d <version>
    git push --delete origin <version>
    ```

## Conclusion

Following these steps will help you create a new release for Capture successfully. Ensure that you have the necessary permissions to push tags and branches to the repository. If you have any questions or need assistance, feel free to reach out to the maintainers.

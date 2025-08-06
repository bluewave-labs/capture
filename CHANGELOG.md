# Changelog for Capture

> This CHANGELOG.md file tracks all released versions and their changes. All releases are automatically published via GitHub Actions and GoReleaser with cryptographic checksums for security verification.

Date-Format: YYYY-MM-DD

## 1.2.0 (2025-06-24)

This release adds support for monitoring network activity and Docker containers. It also includes enhanced API responses with metadata and introduces user-friendly host names (e.g., "Ubuntu 24.04.2 LTS") for improved readability.

* [2de06c3](https://github.com/bluewave-labs/capture/commit/2de06c3cf9acca167d05c6fde52c9c0177dbd6ee) Capture metadata in API Responses (#82)
* [7b98c15](https://github.com/bluewave-labs/capture/commit/7b98c15dfe2ee3feff8f55ba227e44f34f2da686) Issue and pr templates (#86)
* [d7c9c74](https://github.com/bluewave-labs/capture/commit/d7c9c747767fcdb644a3e4d2dc6d0cc6ba9eb9e6) User friendly instructions in README for Quick Start (#93)
* [8161d57](https://github.com/bluewave-labs/capture/commit/8161d57102ee576ab462159acd135a422599048e) build(deps): bump golang.org/x/net from 0.30.0 to 0.38.0 (#84)
* [d7d5824](https://github.com/bluewave-labs/capture/commit/d7d5824c2d53990077ba642777a78c0ed4f5cc10) chore: Add bug report issue template to improve issue tracking (#69)
* [546e533](https://github.com/bluewave-labs/capture/commit/546e533e58342ee8051a0a59a7c9b966a8453cc5) chore: Enhance Dockerfile with additional comments and structure (#88)
* [41283a5](https://github.com/bluewave-labs/capture/commit/41283a5299a3f688b8d354dedb8a275092ccb042) ci: Add codeql.yml (#70)
* [019c1ca](https://github.com/bluewave-labs/capture/commit/019c1ca41fc93f9821f6a297fda84e26efc64d7f) ci: Make go workflow read-only (#74)
* [442bf24](https://github.com/bluewave-labs/capture/commit/442bf24ba9ea6da7a9b24abdbb2763a352707055) docs(security): Update vulnerability reporting guideline (#71)
* [c7ba448](https://github.com/bluewave-labs/capture/commit/c7ba4486c90b577d27afac13d37b9de0036b3b71) feat(api): Update OpenAPI specification to version 1.1.0 (#83)
* [e2580a9](https://github.com/bluewave-labs/capture/commit/e2580a9fc131224382255b1d2053ef7323d163a9) feat(metric): Docker container monitoring (#78)
* [e5ee49d](https://github.com/bluewave-labs/capture/commit/e5ee49d4a5ffdaa2eb82017e65fd7729ab403879) feat(metrics): Add network metrics collection (#67)
* [f0f8fee](https://github.com/bluewave-labs/capture/commit/f0f8fee5fe32d32790b4793e4dd430086f66e0d8) feat: host.prettyname added (#90)
* [592cc72](https://github.com/bluewave-labs/capture/commit/592cc722f8f4c48f1315687eb86007f50814c67a) fix: Correct JSON key for SmartOverallHealthResult in metrics (#87)
* [92de4a2](https://github.com/bluewave-labs/capture/commit/92de4a2aa2582d06ad316b1646450248b3a51d53) fix: Move health check route to the correct position in the handler initialization (#79)

[Full Changelog](https://github.com/bluewave-labs/capture/compare/v1.1.0...v1.2.0)

Contributors: @mertssmnoglu

## 1.1.0 (2025-05-12)

The new Capture release enhances system performance monitoring with features like S.M.A.R.T metrics, disk current read/write stats, iNode usage and a ZFS filtering fix for Debian/Ubuntu systems.

You can access new MacOS pre-built binaries from the [releases page](https://github.com/bluewave-labs/capture/releases).

---

Featured Changes

- [472e7be95987a33dc50d573654dcb1c2f3bee1ab](https://github.com/bluewave-labs/capture/commit/472e7be95987a33dc50d573654dcb1c2f3bee1ab) Feat: Current Read/Write Data (#54) @Br0wnHammer
- [aadedfb99e8afbc5aa34dd3941ba90cc6ce12bcb](https://github.com/bluewave-labs/capture/commit/aadedfb99e8afbc5aa34dd3941ba90cc6ce12bcb) Fix 51 smartctlr metrics od there serve (#53) @Owaiseimdad
- [ef5b2367ae8f10a3f0acb55dbd3211e652ae902e](https://github.com/bluewave-labs/capture/commit/ef5b2367ae8f10a3f0acb55dbd3211e652ae902e) Fix: #46 Inode Usage metrics (#56) @noodlewhale
- [9429bdcae6b8f33e52ef1aa3783098bfe2d311b1](https://github.com/bluewave-labs/capture/commit/9429bdcae6b8f33e52ef1aa3783098bfe2d311b1) feat(logging): Warn users to remember adding endpoint to Checkmate Infrastructure Dashboard (#59) @mertssmnoglu
- [994e4b3188b949604dcadb17ba34941fda75288f](https://github.com/bluewave-labs/capture/commit/994e4b3188b949604dcadb17ba34941fda75288f) fix(disk): Enhance partition filtering logic to include ZFS filesystems #55 (#64) @mertssmnoglu

[Full Changelog](https://github.com/bluewave-labs/capture/compare/v1.0.1...994e4b3188b949604dcadb17ba34941fda75288f)

Contributors: @mertssmnoglu, @Br0wnHammer, @Owaiseimdad, @noodlewhale

## 1.0.1 (2025-02-06)

This release focuses on feature improvements and extending system metrics coverage to enhance functionality and reliability.

> Requires Checkmate >= 2.0
>
> If your version is higher than 2.0, you don't need to upgrade Checkmate.

- [14aff9e0b3deb8771d0aaa9eee61c4c5e023705e](https://github.com/bluewave-labs/capture/commit/14aff9e0b3deb8771d0aaa9eee61c4c5e023705e) feat(main): Add version flag to display application version (#45)
- [93122c59b45c13d1a6914aa30bca2671e1a0336c](https://github.com/bluewave-labs/capture/commit/93122c59b45c13d1a6914aa30bca2671e1a0336c) fix(metric): collect all disk partitions instead of only physical ones (#44)

Contributors: @mertssmnoglu

## 1.0.0 (2024-12-31)

First release of the Capture project.

- [aace2934eb80cbdb8903e76c1c2b57fdfe179454](https://github.com/bluewave-labs/capture/commit/aace2934eb80cbdb8903e76c1c2b57fdfe179454) Merge pull request #36 from bluewave-labs/chore/openapi-specs
- [e984e733f70243bbb05d8231fd9be9f5eea6cdce](https://github.com/bluewave-labs/capture/commit/e984e733f70243bbb05d8231fd9be9f5eea6cdce) Merge pull request #37 from bluewave-labs/ci/lint
- [861c26c340f12bd2b531698d965da59508677a1a](https://github.com/bluewave-labs/capture/commit/861c26c340f12bd2b531698d965da59508677a1a) Merge pull request #38 from bluewave-labs/readme-update
- [a3623ef7c7ae7b3039dd9290bf9aacd7c42d5424](https://github.com/bluewave-labs/capture/commit/a3623ef7c7ae7b3039dd9290bf9aacd7c42d5424) chore(openapi): add openapi 3.0.0 specs for the API
- [8abb8581c45307fac49bc96e2127a4eb4a82ba05](https://github.com/bluewave-labs/capture/commit/8abb8581c45307fac49bc96e2127a4eb4a82ba05) chore(openapi): add security schema and improve example response
- [eb6695df4fd77e8c2e525565b5cc6e25123dcf60](https://github.com/bluewave-labs/capture/commit/eb6695df4fd77e8c2e525565b5cc6e25123dcf60) chore(openapi): remove unimplemented websocket path
- [ba2ab5f6a0184967d88b7fcf5176ff4e561de1fa](https://github.com/bluewave-labs/capture/commit/ba2ab5f6a0184967d88b7fcf5176ff4e561de1fa) chore: Remove unimplemented 'ReadSpeedBytes' and 'WriteSpeedBytes' fields from the DiskData struct
- [4e474b7aff7546c161dab26c4851eb53bb4ff7b9](https://github.com/bluewave-labs/capture/commit/4e474b7aff7546c161dab26c4851eb53bb4ff7b9) ci: Change 'ubuntu-latest' runners to 'ubuntu-22.04' (#40)
- [b00cd259b6fd56e43d1b00360835063527ff3817](https://github.com/bluewave-labs/capture/commit/b00cd259b6fd56e43d1b00360835063527ff3817) ci: add lint.yml
- [7a2fbd97f996b91a50ffed58130fca779453e561](https://github.com/bluewave-labs/capture/commit/7a2fbd97f996b91a50ffed58130fca779453e561) ci: change job name to lint
- [33f51d1a7129fadfadff3dc3bc081abb39cee980](https://github.com/bluewave-labs/capture/commit/33f51d1a7129fadfadff3dc3bc081abb39cee980) docs(README): Clarify how to install and use the Capture
- [0707d258d106452ffc6a033d2699cfd9aa047e89](https://github.com/bluewave-labs/capture/commit/0707d258d106452ffc6a033d2699cfd9aa047e89) docs(README): Describe how to install with 'go install'  and update Environment Variables list
- [bce26b2f194e43b43e955ebf5dd966db5b808530](https://github.com/bluewave-labs/capture/commit/bce26b2f194e43b43e955ebf5dd966db5b808530) fix(lint): Solve all linter warnings and errors
- [acddb720cc9256992f8704f518bfe5e826b6cddd](https://github.com/bluewave-labs/capture/commit/acddb720cc9256992f8704f518bfe5e826b6cddd) fix: remove websocket handler
- [2f8f3f9c18cee756a867a88e5df4279c7124948c](https://github.com/bluewave-labs/capture/commit/2f8f3f9c18cee756a867a88e5df4279c7124948c) refactor(server): improve logging and handle shutdown signals with ease (#39)

Contributors: @mertssmnoglu

# Changelog for Capture

> This CHANGELOG.md file tracks all released versions and their changes. All releases are automatically published via GitHub Actions and GoReleaser with cryptographic checksums for security verification.

Date-Format: YYYY-MM-DD

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

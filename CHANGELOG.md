# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

This file itself is based on [Keep a CHANGELOG](https://keepachangelog.com/en/0.3.0/).

## [Unreleased]

## [0.5.1] - 2023-03-08

### Fixed

- Fix k8s clinet.Delete() argument. ([#53](https://github.com/topolvm/pie/pull/53))

### Contributors

- @cupnes

## [0.5.0] - 2023-03-02

### Changed

- controller: set CronJob.spec.SuccessfulJobsHistoryLimit to 0  ([#46](https://github.com/topolvm/pie/pull/46))
- controller: vary start time of CronJob ([#47](https://github.com/topolvm/pie/pull/47))
- observer: delete Job instead of Pod after create threshold ([#48](https://github.com/topolvm/pie/pull/48))
- Remove ownerReference ([#49](https://github.com/topolvm/pie/pull/49))

### Contributors

- @toshipp
- @daichimukai

## [0.4.1] - 2023-02-10

### Added

- artifacthub ([#39](https://github.com/topolvm/pie/pull/39))

### Changed

- Specify go version by go-version-file ([#37](https://github.com/topolvm/pie/pull/37))
- update a note describing how to maintain go version ([#38](https://github.com/topolvm/pie/pull/38))
- Add Signed-off-by on the bump commit ([#44](https://github.com/topolvm/pie/pull/44))

### Contributors

- @bells17
- @cupnes
- @llamerada-jp
- @toshipp

## [0.4.0] - 2022-12-05

### Added

- add an issue template to update supporting kubernetes ([#26](https://github.com/topolvm/pie/pull/26))
- set dependabot for github actions ([#27](https://github.com/topolvm/pie/pull/27))

### Changed

- Support Kubernetes 1.25. ([#32](https://github.com/topolvm/pie/pull/32))
  - **BREAKING**: Support for Kubernetes 1.22 has been dropped.
- Replace event reconciler with pod reconciler to reduce api-server load ([#33](https://github.com/topolvm/pie/pull/33))

### Contributors

- @peng225
- @toshipp

## [0.3.1] - 2022-11-21

### Changed

- add a command to list the relevant PRs in the release procedure. ([#24](https://github.com/topolvm/pie/pull/24))

### Fixed

- fix the occasion of getNodeNameAndStorageClass error message ([#25](https://github.com/topolvm/pie/pull/25))
- fixed to work garbage collection correctly  ([#28](https://github.com/topolvm/pie/pull/28))

### Contributors

- @peng225
- @cupnes
- @toshipp

## [0.3.0] - 2022-11-10

### Added

- add project-bot workflow and issue templates (#19)

### Changed

- enhance node selector to fix jobs or pods pending (#21)

### Contributors

- @cupnes
- @toshipp

## [0.2.0] - 2022-11-02

### Changed

- change the prefix of metric names to "pie". (#15)
  - **BREAKING**: Metrics names have been changed.

### Contributors

- @peng225

## [0.1.1] - 2022-10-26

### Changed

- specify user in Dockerfile (#12)

### Fixed

- remove unnecessary argument (#11)

### Contributors

- @cupnes

## [0.1.0] - 2022-10-24

This is the first release.

[Unreleased]: https://github.com/topolvm/pie/compare/v0.5.1...HEAD
[0.5.1]: https://github.com/topolvm/pie/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/topolvm/pie/compare/v0.4.1...v0.5.0
[0.4.1]: https://github.com/topolvm/pie/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/topolvm/pie/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/topolvm/pie/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/topolvm/pie/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/topolvm/pie/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/topolvm/pie/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/topolvm/pie/compare/4b825dc642cb6eb9a060e54bf8d69288fbee4904...v0.1.0

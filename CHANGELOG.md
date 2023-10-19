# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

This file itself is based on [Keep a CHANGELOG](https://keepachangelog.com/en/0.3.0/).

**Note: See the [release notes](https://github.com/topolvm/pie/releases) for changes after v0.7.2.**

## [Unreleased]

## [0.7.2] - 2023-08-17

### Added

- add metrics ([#84](https://github.com/topolvm/pie/pull/84))

### Contributors

- @toshipp

## [0.7.1] - 2023-08-09

### Changed

- add Ryotaro Banno to owners ([#77](https://github.com/topolvm/pie/pull/77))
- Use dependabot grouping feature ([#78](https://github.com/topolvm/pie/pull/78))
- Migrate kubebuilder project to v4 ([#79](https://github.com/topolvm/pie/pull/79))
- Rename controller package to metrics ([#80](https://github.com/topolvm/pie/pull/80))

### Contributors

- @llamerada-jp
- @toshipp

## [0.7.0] - 2023-07-07

### Changed

- support kubernetes 1.27 ([#70](https://github.com/topolvm/pie/pull/70))
  - **BREAKING**: Kubernetes 1.24 is not supported.
- Add an item to the check list for Kubernetes upgrade to ensure that t… ([#71](https://github.com/topolvm/pie/pull/71))
- Change builder image version to golang:1.20-bullseye ([#74](https://github.com/topolvm/pie/pull/74))

### Contributors

- @cupnes
- @peng225

## [0.6.1] - 2023-05-18

### Added

- Add pprof function ([#66](https://github.com/topolvm/pie/pull/66))

### Changed

- Embed stderr from running command in error ([#65](https://github.com/topolvm/pie/pull/65))

### Fixed

- Fix map growing ([#67](https://github.com/topolvm/pie/pull/67))

### Contributors

- @toshipp

## [0.6.0] - 2023-04-13

### Added

- add a workflow job to check the do-not-merge label ([#56](https://github.com/topolvm/pie/pull/56))
- add documentation of pie stop and start operations ([#57](https://github.com/topolvm/pie/pull/57))

### Changed

- Bump actions/setup-go from 3 to 4 ([#59](https://github.com/topolvm/pie/pull/59))
- Update supporting Kubernetes (1.26) ([#60](https://github.com/topolvm/pie/pull/60))
  - **BREAKING**: Kubernetes 1.23 is not supported.
- change the spec of metrics ([#61](https://github.com/topolvm/pie/pull/61))
  - **BREAKING**: The metrics `pie_create_probe_fast_total` and `pie_create_probe_slow_total` are deleted. Use `pie_create_probe_total`.

### Contributors

- @cupnes
- @llamerada-jp
- @peng225
- @toshipp

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

[Unreleased]: https://github.com/topolvm/pie/compare/v0.7.2...HEAD
[0.7.2]: https://github.com/topolvm/pie/compare/v0.7.1...v0.7.2
[0.7.1]: https://github.com/topolvm/pie/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/topolvm/pie/compare/v0.6.1...v0.7.0
[0.6.1]: https://github.com/topolvm/pie/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/topolvm/pie/compare/v0.5.1...v0.6.0
[0.5.1]: https://github.com/topolvm/pie/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/topolvm/pie/compare/v0.4.1...v0.5.0
[0.4.1]: https://github.com/topolvm/pie/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/topolvm/pie/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/topolvm/pie/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/topolvm/pie/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/topolvm/pie/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/topolvm/pie/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/topolvm/pie/compare/4b825dc642cb6eb9a060e54bf8d69288fbee4904...v0.1.0

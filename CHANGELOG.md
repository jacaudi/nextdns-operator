# Changelog

## [0.17.0](https://github.com/jacaudi/nextdns-operator/compare/v0.16.1...v0.17.0) (2026-04-08)

### Features

* add Gateway spec.infrastructure support ([#114](https://github.com/jacaudi/nextdns-operator/issues/114)) ([4edd518](https://github.com/jacaudi/nextdns-operator/commit/4edd5189d53e65c6656bcf77b791636eaad64114)), closes [#113](https://github.com/jacaudi/nextdns-operator/issues/113)

## [0.16.1](https://github.com/jacaudi/nextdns-operator/compare/v0.16.0...v0.16.1) (2026-04-08)

### Bug Fixes

* GatewayClass redesign - operator as consumer, not controller ([#112](https://github.com/jacaudi/nextdns-operator/issues/112)) ([24f4c8a](https://github.com/jacaudi/nextdns-operator/commit/24f4c8a592dc4e759dbdb7f18cf863dc19440359))

## [0.16.0](https://github.com/jacaudi/nextdns-operator/compare/v0.15.4...v0.16.0) (2026-04-07)

### Bug Fixes

* address final review findings in gateway helpers ([d47eccf](https://github.com/jacaudi/nextdns-operator/commit/d47eccf605eb56d8851f191bc74651c97d39659d))
* auto-sync CRDs to Helm chart in manifests task ([00a0a93](https://github.com/jacaudi/nextdns-operator/commit/00a0a9374d4158f142f47331e157f4990966b89c))
* ignore .worktrees/ directory in gitignore ([db659aa](https://github.com/jacaudi/nextdns-operator/commit/db659aa0f6229c5b5c08a852c9ebed7d87446050))
* sync Helm chart CRDs with generated CRDs ([71ce048](https://github.com/jacaudi/nextdns-operator/commit/71ce048df6ede34268369b7734b81c36cf285e37))


### Features

* add Gateway API CRD detection and GatewayClass startup creation ([09472e0](https://github.com/jacaudi/nextdns-operator/commit/09472e004875a0782b852246c6e3939530016c6b))
* add Gateway API RBAC markers, update Helm chart, add sample CR ([8ca8136](https://github.com/jacaudi/nextdns-operator/commit/8ca813697c8a22b00a8780aafd9eab680cff0fae))
* add gateway validation (mutual exclusivity and CRD detection) ([43f8292](https://github.com/jacaudi/nextdns-operator/commit/43f8292002aa2d88c8f6fcd891de050311bf0aa9))
* add GatewayConfig and GatewayAddress API types ([726cc19](https://github.com/jacaudi/nextdns-operator/commit/726cc19fafca334a890b52376969403364a4ef98))
* add sigs.k8s.io/gateway-api dependency ([24babc6](https://github.com/jacaudi/nextdns-operator/commit/24babc67ee91e8e7f6a909e0899474ec9c289a18))
* implement gateway, TCPRoute, and UDPRoute reconciliation helpers ([4a11746](https://github.com/jacaudi/nextdns-operator/commit/4a11746617246ebef69935522288b0d4ffeb55fa))
* wire gateway reconciliation into main loop and force ClusterIP when gateway is set ([986e8ca](https://github.com/jacaudi/nextdns-operator/commit/986e8cac32223384da70a50a247b5be9d5c15f5e))
* wire gateway status updates into reconcile loop and add conditional watches ([1aa57ee](https://github.com/jacaudi/nextdns-operator/commit/1aa57ee4b99854550e38f027ad71cfde11612ec9))

## [0.15.4](https://github.com/jacaudi/nextdns-operator/compare/v0.15.3...v0.15.4) (2026-04-07)

### Bug Fixes

* use new-release-version-v output for v-prefixed versions ([db3a2a1](https://github.com/jacaudi/nextdns-operator/commit/db3a2a1e7717d0c014f3e5e3ea6829008e61417f))

## [0.15.3](https://github.com/jacaudi/nextdns-operator/compare/v0.15.2...v0.15.3) (2026-04-06)

### Bug Fixes

* add v prefix to container image tag ([85f728c](https://github.com/jacaudi/nextdns-operator/commit/85f728c475fd264afdd7f3b0302f284ef3d5e6d4))

## [0.15.2](https://github.com/jacaudi/nextdns-operator/compare/v0.15.1...v0.15.2) (2026-04-06)

### Bug Fixes

* add v prefix to helm chart OCI tag ([3b6bf2c](https://github.com/jacaudi/nextdns-operator/commit/3b6bf2c9177f800746cd8681a28c6d4ad97eb7de))

## [0.15.1](https://github.com/jacaudi/nextdns-operator/compare/v0.15.0...v0.15.1) (2026-04-06)

### Bug Fixes

* fall back to linkedIP.servers when setup.ipv4 is empty ([#100](https://github.com/jacaudi/nextdns-operator/issues/100)) ([f8a6725](https://github.com/jacaudi/nextdns-operator/commit/f8a67253b1db315d1e62029c633196a2a50f3a52))

## 0.15.0 (2026-04-06)

#### Feature

* fix retention sync, add status.setup, profile-specific CoreDNS IPs (#104) ([f255102d](https://github.com/jacaudi/nextdns-operator/commit/f255102d887469a77055ca0c1259c6efdce91066))

#### CI

* add manual release gate and promote skip-update logs to INFO (#102) ([793ac805](https://github.com/jacaudi/nextdns-operator/commit/793ac80536f5cac28eceb3c5ea120e1b11759e41))


## 0.14.11 (2026-04-05)

#### Bug Fixes

* **ci:** bump github-actions to v0.15.3 ([5e2f0aef](https://github.com/jacaudi/nextdns-operator/commit/5e2f0aefef658ec5b423c487bf0e51a85031c884))

#### Chores

* **changelog:** update for v0.14.10 [skip ci] ([0e2fccec](https://github.com/jacaudi/nextdns-operator/commit/0e2fccece3c68d0277ed8aaa2b3a24e30ac51e1e))


## 0.14.10 (2026-04-05)

#### Bug Fixes

* **docs:** clean up changelog noise and remove stale header ([8e61a4ce](https://github.com/jacaudi/nextdns-operator/commit/8e61a4ce4cb8a698553b6ac376c15776aa22906f))

#### Chores

* **changelog:** update for v0.14.9 [skip ci] ([6879879b](https://github.com/jacaudi/nextdns-operator/commit/6879879be744066a343ab32f7849822257ca2058))


## 0.14.9 (2026-04-05)

#### Bug Fixes

* **ci:** bump github-actions to v0.15.2 ([dcf917a5](https://github.com/jacaudi/nextdns-operator/commit/dcf917a510f17ed59710d6946f9906a242875261))


## 0.14.8 (2026-04-04)

#### Bug Fixes

* **ci:** disable emojis in changelog generator ([0ce90c00](https://github.com/jacaudi/nextdns-operator/commit/0ce90c007a5550ade66301798db2a681d03ca6db))


## 0.14.7 (2026-04-04)

#### Bug Fixes

* **ci:** align semrelrc config with github-actions template ([b1254e76](https://github.com/jacaudi/nextdns-operator/commit/b1254e76e095aa278d65a26d0e0b7cb5b9313863))


## 0.14.6 (2026-04-04)

#### Bug Fixes

* **docs:** add missing v0.14.4 and v0.14.5 changelog entries ([e0d64d73](https://github.com/jacaudi/nextdns-operator/commit/e0d64d73))

---

## [v0.14.5](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.14.5) - 2026-04-04

- [`082ceca`](https://github.com/jacaudi/nextdns-operator/commit/082ceca) fix: resolve lint failures from v0.15.1 upgrade

## [v0.14.4](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.14.4) - 2026-04-04

- [`bde175d`](https://github.com/jacaudi/nextdns-operator/commit/bde175d) fix(ci): upgrade github-actions to v0.15.1 with component- prefix rename

## [v0.14.3](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.14.3) - 2026-04-03

- [`3289ce1`](https://github.com/jacaudi/nextdns-operator/commit/3289ce1e22acdc1d57e8c0bd096205189f5769ef) fix(ci): configure go-semantic-release with changelog and GitHub provider

## [v0.14.2](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.14.2) - 2026-04-03

- [`166fb62`](https://github.com/jacaudi/nextdns-operator/commit/166fb624bdeaad4ac6d3cc1d0dafb8dd08a2d337) fix(ci): bump github-actions to v0.14.2 for changelog commit-back fix

## [v0.14.1](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.14.1) - 2026-04-03

- [`8078893`](https://github.com/jacaudi/nextdns-operator/commit/8078893d0fe5dabe9e6ee5ce5ec3af5bef5384bc) fix(docs): simplify README features section
- [`ca55e29`](https://github.com/jacaudi/nextdns-operator/commit/ca55e29cbfa3d604dbfce559551334f47ca6cb1f) ci: enable automatic CHANGELOG.md generation via go-semantic-release

## [v0.14.0](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.14.0) - 2026-04-03

- [`6d16a82`](https://github.com/jacaudi/nextdns-operator/commit/6d16a8223370485b9d6bebf3f5b6d4ddd71e9bd9) feat: CoreDNS enhancements - deviceName warning, ServiceMonitor removal, PDB support (#98)
- [`ace1005`](https://github.com/jacaudi/nextdns-operator/commit/ace1005952599100b861b1f8d162b34f2730722b) chore: remove NodePort as supported CoreDNS service type (#97)

## [v0.13.3](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.13.3) - 2026-04-01

- [`69ae291`](https://github.com/jacaudi/nextdns-operator/commit/69ae29144a6c602c282e08d4333cb4483290bb42) fix: validate ProfileID before building CoreDNS Corefile (#96)
- [`e1a99e4`](https://github.com/jacaudi/nextdns-operator/commit/e1a99e4675279b7ae836fc3599a70d9b54ee1a73) chore: migrate from Makefile to Taskfile (#89)

## [v0.13.2](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.13.2) - 2026-03-29

- [`4fe5d1b`](https://github.com/jacaudi/nextdns-operator/commit/4fe5d1b7b7ac26648800fa59f3bb46a7eac1c07d) fix: add events RBAC for leader election announcements

## [v0.13.1](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.13.1) - 2026-03-29

- [`c442621`](https://github.com/jacaudi/nextdns-operator/commit/c44262112af9e5aa895374fffcea34388fd1c247) fix: prevent reconcile loop from unconditional status updates (#88)
- [`acbb905`](https://github.com/jacaudi/nextdns-operator/commit/acbb90597d2b17fc6caff95808033fdd15d47d73) ci: rename docker to container and parallelize container+helm builds (#86)

## [v0.13.0](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.13.0) - 2026-03-29

- [`a2d4bf3`](https://github.com/jacaudi/nextdns-operator/commit/a2d4bf3ecc80fd9189822dce0f36e052088a0c26) feat: add BAV (Bypass Age Verification) to settings (#85)

## [v0.12.1](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.12.1) - 2026-03-28

- [`1566a9a`](https://github.com/jacaudi/nextdns-operator/commit/1566a9a73acee8e23a929c374c5d89deba6a4bf0) fix(deps): update module github.com/jacaudi/nextdns-go to v0.13.0

## [v0.12.0](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.12.0) - 2026-03-28

- [`3fa2e14`](https://github.com/jacaudi/nextdns-operator/commit/3fa2e14ea86f34b89f2a9bf4d5397ac0df2c29e2) feat: observe mode completeness - setup, parental control, logs.location (#84)

## [v0.11.3](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.11.3) - 2026-03-27

- [`21aa93d`](https://github.com/jacaudi/nextdns-operator/commit/21aa93d180a650d1a77452879864c7371dedd989) fix: use real API fingerprint instead of hardcoded DNS endpoint (#82)

## [v0.11.2](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.11.2) - 2026-03-27

- [`ce6ee25`](https://github.com/jacaudi/nextdns-operator/commit/ce6ee251d527aaa5446f84ea4acad3a539a65452) fix: read and invert logs.drop fields in observe mode (#81)

## [v0.11.1](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.11.1) - 2026-03-27

- [`e9128cf`](https://github.com/jacaudi/nextdns-operator/commit/e9128cf04ee48fed7190767e2fd800e6c3c162f2) fix: clamp formatRetentionString to valid CRD enum values (#72) (#73)
- [`9027202`](https://github.com/jacaudi/nextdns-operator/commit/9027202865ff8361b790006c7bf9553b122bea4a) docs: add implementation plan for cross-namespace secrets

## [v0.11.0](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.11.0) - 2026-03-26

- [`6641ce1`](https://github.com/jacaudi/nextdns-operator/commit/6641ce19b00e912904d6c807e5543300d32989c4) feat: support cross-namespace secret references in credentialsRef (#71)
- [`a1f77f7`](https://github.com/jacaudi/nextdns-operator/commit/a1f77f7bb0fdda7e9057a0d8f94f7d0687fda6da) docs: update documentation for current feature set

## [v0.10.1](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.10.1) - 2026-03-26

- [`f9e8088`](https://github.com/jacaudi/nextdns-operator/commit/f9e80885606cec3a85eaba9bb5f8376a3861101d) fix(deps): update kubernetes-client-libraries to v0.35.3

## [v0.10.0](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.10.0) - 2026-03-26

- [`eaacfc5`](https://github.com/jacaudi/nextdns-operator/commit/eaacfc5c71436cd0d196095961849a0ab425edab) feat: remove deprecated ConfigMap Import feature (#69)

## [v0.9.0](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.9.0) - 2026-03-26

- [`ff4af50`](https://github.com/jacaudi/nextdns-operator/commit/ff4af5072fc00356023d48096afa96938332ef29) feat: observe-only mode for existing NextDNS profiles (#62)

## [v0.8.3](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.8.3) - 2026-03-26

- [`77a2c47`](https://github.com/jacaudi/nextdns-operator/commit/77a2c47e066e7f21ca32ac9026fae5c84a3fabec) fix: address minor code review findings (M1-M8) (#67)

## [v0.8.2](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.8.2) - 2026-03-24

- [`667bb11`](https://github.com/jacaudi/nextdns-operator/commit/667bb112d8d2e74bfa2ea996ebfabfbdf0b0461a) fix: address important code review findings (I1-I8) (#66)

## [v0.8.1](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.8.1) - 2026-03-24

- [`c683552`](https://github.com/jacaudi/nextdns-operator/commit/c683552dbfa6a4a0ad41f5de28c50657b8aa5336) fix: sync all spec fields to NextDNS API in managed mode (C1, C2) (#65)
- [`1f46f87`](https://github.com/jacaudi/nextdns-operator/commit/1f46f87de2fb6e27c2e2e0d8203a2ab500259ff0) ci: bump jacaudi/github-actions to v0.14.1
- [`3b855a0`](https://github.com/jacaudi/nextdns-operator/commit/3b855a096d62f13220ecb160190b33778583dbb9) docs: update WORKFLOWS.md to reflect trivy removal
- [`8591960`](https://github.com/jacaudi/nextdns-operator/commit/859196011c5f3ba1e720eb40a7be90ef4cf5cd24) ci: remove trivy image scanning
- [`de5739f`](https://github.com/jacaudi/nextdns-operator/commit/de5739f80928c4a0067cc8dcf3453f53a63d5c29) chore(config): migrate config .github/renovate.json (#61)

## [v0.8.0](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.8.0) - 2026-03-18

- [`34738c1`](https://github.com/jacaudi/nextdns-operator/commit/34738c1cb8e64b900089c57abafb41b1afec1a9b) feat: add first-class Multus CNI support for secondary IPs (#58)

## [v0.7.3](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.7.3) - 2026-03-18

- [`64586fd`](https://github.com/jacaudi/nextdns-operator/commit/64586fd7e0562e17de89aec3db1848731c412f2a) fix: regenerate CRDs for controller-tools v0.20.1 and add postUpgradeTasks
- [`07cc455`](https://github.com/jacaudi/nextdns-operator/commit/07cc45530edd472c743985a5bfbf052c3a339e49) chore(deps): update module sigs.k8s.io/controller-tools to v0.20.1

## [v0.7.2](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.7.2) - 2026-03-15

- [`1322841`](https://github.com/jacaudi/nextdns-operator/commit/132284199ba951816ff9d4e0dd767983452f9dd6) fix(deps): update kubernetes-client-libraries
- [`6016238`](https://github.com/jacaudi/nextdns-operator/commit/6016238caf910ef62b03bd2129d0722017da252a) chore(deps): update jacaudi/github-actions action to v0.14.0
- [`096fade`](https://github.com/jacaudi/nextdns-operator/commit/096fade850060ac40414e1d43c188267372a3823) chore(deps): update github actions
- [`f1ee04e`](https://github.com/jacaudi/nextdns-operator/commit/f1ee04ec7df64d0fa5ec51fbc5beb4eee0010842) chore(deps): update dependency go to v1.26.1
- [`444049d`](https://github.com/jacaudi/nextdns-operator/commit/444049d5622a01ce51f2d301b54be751b5ac42d7) chore(deps): update golang docker tag to v1.26.1

## [v0.7.1](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.7.1) - 2026-02-18

- [`8a04d51`](https://github.com/jacaudi/nextdns-operator/commit/8a04d51c5ed4ae6cfc27c736caf8f823ea353c9b) fix(deps): update kubernetes-client-libraries to v0.35.1
- [`25e3e3f`](https://github.com/jacaudi/nextdns-operator/commit/25e3e3fdda8ec5b20fe367887bef39d0a4bb4c39) chore(deps): update github actions
- [`16b00b9`](https://github.com/jacaudi/nextdns-operator/commit/16b00b999824967531fa4f10c46a9050a502c9ed) chore(deps): update dependency go to v1.26.0
- [`704042f`](https://github.com/jacaudi/nextdns-operator/commit/704042f12449eea6ae271537fd8bbe28cfbb1449) chore(deps): update golang docker tag to v1.26.0
- [`6b1cbcc`](https://github.com/jacaudi/nextdns-operator/commit/6b1cbcc910c1e8caf2de45b16239157b6d8e8da2) chore(deps): update actions/setup-go action to v6 (#42)
- [`8b68e34`](https://github.com/jacaudi/nextdns-operator/commit/8b68e34ef1dcabc04b6fa80d702c9b1ed703adb4) chore(deps): update dependency go to v1.25.7 (#9)
- [`91a9a35`](https://github.com/jacaudi/nextdns-operator/commit/91a9a3504514ace97b87f28292ac941a68e89df2) chore(deps): update golang docker tag to v1.25.7 (#8)

## [v0.7.0](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.7.0) - 2026-02-07

- [`8566d3a`](https://github.com/jacaudi/nextdns-operator/commit/8566d3a98470fd3e812c7cb41d35d521d8bd0efc) feat: import profile configuration from ConfigMap JSON (#45)

## [v0.6.0](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.6.0) - 2026-02-07

- [`486595f`](https://github.com/jacaudi/nextdns-operator/commit/486595fdbd760fb541dd5f179e59b71418a8711f) feat: add domain override support to NextDNSCoreDNS (#44)

## [v0.5.3](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.5.3) - 2026-02-04

- [`a856967`](https://github.com/jacaudi/nextdns-operator/commit/a856967f73d5221c4e8257b0fedc8352dd5db093) fix(coredns): remove fallback protocol support to fix forward syntax (#40)
- [`9bf08aa`](https://github.com/jacaudi/nextdns-operator/commit/9bf08aafe26d1bf9261d62d82952796000ef1e31) ci: add coredns package to CI/CD test coverage

## [v0.5.2](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.5.2) - 2026-02-04

- [`b5ce95b`](https://github.com/jacaudi/nextdns-operator/commit/b5ce95b00783427c6561b9634afd741f7e20f7bc) fix: CoreDNS security context, image update, and Helm CRD sync (#38)

## [v0.5.1](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.5.1) - 2026-02-04

- [`772deeb`](https://github.com/jacaudi/nextdns-operator/commit/772deeb3725a6d2d4297df98c032193d3c97c79a) fix(rbac): use plural resource name for NextDNSCoreDNS (#35)

## [v0.5.0](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.5.0) - 2026-02-04

- [`9d20551`](https://github.com/jacaudi/nextdns-operator/commit/9d20551010aa73ec0cbaacbcadb49e3e6f3104c1) feat: Add NextDNSCoreDNS CRD for deploying CoreDNS with NextDNS upstream (#33)
- [`5923807`](https://github.com/jacaudi/nextdns-operator/commit/592380738cc609efb004f650922347173bca5ed0) Merge pull request #31 from jacaudi/docs/acknowledgements
- [`b46250b`](https://github.com/jacaudi/nextdns-operator/commit/b46250b6dbeed83ce9242ed7b17cb73eb3b8abb8) docs: add acknowledgements section
- [`adfedfe`](https://github.com/jacaudi/nextdns-operator/commit/adfedfe4d1edb50787ed434535d0a3d68598b281) Merge pull request #29 from jacaudi/renovate/github-actions
- [`6c8c316`](https://github.com/jacaudi/nextdns-operator/commit/6c8c316499f3def32dde748356f0b15a2b29db8e) chore(deps): update jacaudi/github-actions action to v0.8.0

## [v0.4.3](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.4.3) - 2026-02-01

- [`f1aed40`](https://github.com/jacaudi/nextdns-operator/commit/f1aed40a6e500c55a3b982075eab6f29011d1ac5) Merge pull request #30 from jacaudi/fix/list-controller-rbac-group
- [`92cddf0`](https://github.com/jacaudi/nextdns-operator/commit/92cddf09be3b1a991801cd2b46db608bf5458424) fix(rbac): use correct API group for list controllers

## [v0.4.2](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.4.2) - 2026-02-01

- [`49892b7`](https://github.com/jacaudi/nextdns-operator/commit/49892b73edf0c5db8072f8a6e62c45fae72d45db) fix(rbac): add lease permissions for leader election

## [v0.4.1](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.4.1) - 2026-02-01

- [`20da18b`](https://github.com/jacaudi/nextdns-operator/commit/20da18b219387eae0aa25365ca220976a836f9ea) fix(rbac): add ConfigMap permissions to Helm chart (#27)
- [`3029bbe`](https://github.com/jacaudi/nextdns-operator/commit/3029bbed19c6b660eff5f6d3f90e418030844a39) docs: add ConfigMap export documentation

## [v0.4.0](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.4.0) - 2026-01-31

- [`4c1f85f`](https://github.com/jacaudi/nextdns-operator/commit/4c1f85f2b4b587d33045f13b8662ca85613a3138) Merge pull request #26 from jacaudi/feature/configmap-connection-details
- [`840d289`](https://github.com/jacaudi/nextdns-operator/commit/840d289b1019aae116ca06850defd1c987f8bdc8) test(controller): add test for ConfigMap watch trigger
- [`c5c61a2`](https://github.com/jacaudi/nextdns-operator/commit/c5c61a219c01d92fa9038ab9ba588981cf6eef68) feat(controller): watch ConfigMaps for re-reconciliation
- [`43b3200`](https://github.com/jacaudi/nextdns-operator/commit/43b3200adba95293961f8cbb5cad83169f53c8ce) feat(controller): implement ConfigMap creation for connection details
- [`680f173`](https://github.com/jacaudi/nextdns-operator/commit/680f173667193cfe4618c92038e8a3af55a896a3) test(controller): add tests for ConfigMap creation
- [`9243485`](https://github.com/jacaudi/nextdns-operator/commit/92434854a109d6aa69451be09b6b0a7fb92effbd) feat(rbac): add ConfigMap permissions for profile controller
- [`a9ae704`](https://github.com/jacaudi/nextdns-operator/commit/a9ae704e44f34d72f041e00fa149d7662ad1d969) feat(api): add ConfigMapRef field to NextDNSProfile spec
- [`b8b3a60`](https://github.com/jacaudi/nextdns-operator/commit/b8b3a609f6c6cac01c5d6a1c692fcede275047a8) fix: rename testing.go to testing_helpers_test.go

## [v0.3.0](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.3.0) - 2026-01-31

- [`dd2a6ee`](https://github.com/jacaudi/nextdns-operator/commit/dd2a6eeeb08d0d66dcad0e3273c38677dbae2aec) Merge pull request #24 from jacaudi/feature/sdk-upgrade-v0.11.0
- [`34466ad`](https://github.com/jacaudi/nextdns-operator/commit/34466ad02028c9cb05c6d3082d25cad2603febd0) test(nextdns): add tests for individual list operations
- [`5fe2c08`](https://github.com/jacaudi/nextdns-operator/commit/5fe2c08b04ebe9008d12da44ee4738a63e0ea0e5) test(nextdns): add HasErrorCode test coverage
- [`58e5f1f`](https://github.com/jacaudi/nextdns-operator/commit/58e5f1f3d88529aa06ee25383d7ffa1440858971) feat(nextdns): implement TLD and privacy native operations
- [`af5b8d4`](https://github.com/jacaudi/nextdns-operator/commit/af5b8d459e15601a7b61188fb8c2cc5061ec4e08) feat(nextdns): implement individual allowlist and denylist operations
- [`a49ed6c`](https://github.com/jacaudi/nextdns-operator/commit/a49ed6c4ca3a238aa7307a73343e3406a3596dde) feat(nextdns): add individual list operations to interface
- [`e67ef4d`](https://github.com/jacaudi/nextdns-operator/commit/e67ef4db2be3f157cd6c97fd093bb9cad3777d0f) feat(nextdns): add error helper utilities
- [`c862a95`](https://github.com/jacaudi/nextdns-operator/commit/c862a95e622e1b25be1c814319428127355fc5cc) chore(deps): upgrade nextdns-go SDK to v0.11.0
- [`3cd5c80`](https://github.com/jacaudi/nextdns-operator/commit/3cd5c80531e4f2aef5c3985d0968f233197d0232) chore: add .worktrees to gitignore
- [`8a698e7`](https://github.com/jacaudi/nextdns-operator/commit/8a698e742f9d597702f68786086aa82aa5bb8279) chore(deps): migrate to shared renovate config (#23)

## [v0.1.3](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.1.3) - 2026-01-30

- [`bb14243`](https://github.com/jacaudi/nextdns-operator/commit/bb142436d0e722a564858def809e6425abac84f3) fix: replace deprecated Requeue with RequeueAfter
- [`58d9eda`](https://github.com/jacaudi/nextdns-operator/commit/58d9eda2d03e0092adbaa4c263fa95bb0751378a) chore(deps): update module sigs.k8s.io/controller-runtime to v0.23.1
- [`4cd7454`](https://github.com/jacaudi/nextdns-operator/commit/4cd745452f51b67d446523dcd898422d6bdeaefb) chore(renovate): group kubernetes dependencies together
- [`a10103d`](https://github.com/jacaudi/nextdns-operator/commit/a10103d7ef051b0910ab2c97dfc05c1303909bd3) chore: configure Renovate dashboard and go mod tidy

## [v0.1.2](https://github.com/jacaudi/nextdns-operator/releases/tag/v0.1.2) - 2026-01-30

- [`8c8f104`](https://github.com/jacaudi/nextdns-operator/commit/8c8f104e5da6555aa294cc6d7586444a9b8afc2f) ci: use native multi-arch builds for PR validation
- [`74e7f3b`](https://github.com/jacaudi/nextdns-operator/commit/74e7f3b50e35e15f1dbf592106e74f965bb851d4) chore(ci): update PR workflow to github-actions v0.5.2
- [`134314d`](https://github.com/jacaudi/nextdns-operator/commit/134314dbeb9c020b8d1d063ff5c244e7f39648d4) refactor(controller): update resolvedLists to use DomainEntry type
- [`29a0448`](https://github.com/jacaudi/nextdns-operator/commit/29a0448bf43cae4f82e7ab0d875f529953061a04) refactor(client): update SyncAllowlist to accept DomainEntry with active state
- [`22ede0b`](https://github.com/jacaudi/nextdns-operator/commit/22ede0b6ac9e6cc7c23f76bf6aea3cf07254e10e) refactor(client): update SyncDenylist to accept DomainEntry with active state
- [`feb3860`](https://github.com/jacaudi/nextdns-operator/commit/feb38601d34a9178aea55ac1e6ae0ce1357a5ce4) refactor(client): add DomainEntry type for active state tracking
- [`bab0743`](https://github.com/jacaudi/nextdns-operator/commit/bab07431332b724d7e4682fd991bb3f3f95b8457) fix(rbac): add full CRUD permissions for list CRDs

# Changelog

## [1.9.0](https://github.com/soerenschneider/sc-agent/compare/v1.8.0...v1.9.0) (2025-02-21)


### Features

* make sure data is always written to newly added pki backends ([9e6f6b1](https://github.com/soerenschneider/sc-agent/commit/9e6f6b196f027974d0d13800f19369c5b6606da2))


### Bug Fixes

* **deps:** bump github.com/caarlos0/env/v11 from 11.2.2 to 11.3.1 ([cac96fe](https://github.com/soerenschneider/sc-agent/commit/cac96fe6cd0bed52a90c41252c68407a89f775af))
* **deps:** bump github.com/go-playground/validator/v10 ([12e9543](https://github.com/soerenschneider/sc-agent/commit/12e9543ddb720182089df21d689dc87a281dc90d))
* **deps:** bump github.com/prometheus-community/pro-bing ([d4c5217](https://github.com/soerenschneider/sc-agent/commit/d4c52178ced38c5cb07b175db80584712e072ecb))
* **deps:** bump github.com/prometheus/common from 0.60.1 to 0.62.0 ([9fd6ab8](https://github.com/soerenschneider/sc-agent/commit/9fd6ab8b80f7007d4cd7949d3bd09e7a0da7cb6c))
* **deps:** bump github.com/zcalusic/sysinfo from 1.1.2 to 1.1.3 ([0a710d7](https://github.com/soerenschneider/sc-agent/commit/0a710d7900b51b986dcf0262cb2c4685923bfee7))
* **deps:** bump golang from 1.23.2 to 1.23.4 ([45c7064](https://github.com/soerenschneider/sc-agent/commit/45c7064617d732a3bfd5578f9e98c11b987bac11))
* **deps:** bump golang from 1.23.4 to 1.24.0 ([5714c8d](https://github.com/soerenschneider/sc-agent/commit/5714c8d87046fecb6fe1125b47628bca6ee34a23))
* **deps:** bump golang.org/x/crypto from 0.28.0 to 0.33.0 ([911ce06](https://github.com/soerenschneider/sc-agent/commit/911ce06512c21aca9eb6d27100bf4a8fe0ca0010))
* **deps:** bump golang.org/x/term from 0.25.0 to 0.28.0 ([5e87f77](https://github.com/soerenschneider/sc-agent/commit/5e87f778ed7471bce4270519d5b6ffffb42095d2))
* **http_replication:** only proceed if response's status code is 2xx ([544ad13](https://github.com/soerenschneider/sc-agent/commit/544ad130d83d1f3e3a23090767a00adac5b518d6))
* **http_replication:** only replicate item if response is not empty ([3ee51d8](https://github.com/soerenschneider/sc-agent/commit/3ee51d81bdf62caefe9bd52d40ba7c9778378efb))

## [1.8.0](https://github.com/soerenschneider/sc-agent/compare/v1.7.0...v1.8.0) (2024-11-08)


### Features

* **http_replication:** notice when file has changed on disk outside of sc-agent ([a82b100](https://github.com/soerenschneider/sc-agent/commit/a82b1002cba906f3a9c2d901c926ce7068be273d))
* **secret_replication:** notice when file has changed on disk outside of sc-agent ([c1d1999](https://github.com/soerenschneider/sc-agent/commit/c1d1999a2b8ed76ec8a446f5aa24eac2decf916e))


### Bug Fixes

* **deps:** bump github.com/prometheus/common from 0.60.0 to 0.60.1 ([ff808d3](https://github.com/soerenschneider/sc-agent/commit/ff808d328b2f23de6c8c9e09b6308829175b0d3b))
* fix lint issues ([8050d37](https://github.com/soerenschneider/sc-agent/commit/8050d37592f23699ab1e4d0ee117a19648da5808))

## [1.7.0](https://github.com/soerenschneider/sc-agent/compare/v1.6.0...v1.7.0) (2024-10-27)


### Features

* ability to define fall-back healthiness on continuous errors ([0a0dc4b](https://github.com/soerenschneider/sc-agent/commit/0a0dc4bd32c41839ae79ec4b6faef6f3d8cc2d7e))
* add 'component' logging key ([82ee667](https://github.com/soerenschneider/sc-agent/commit/82ee6677c88d62e820ad8c6c20d6efa245cd0a40))
* add 'dry-run' option for conditional-reboot ([85940ec](https://github.com/soerenschneider/sc-agent/commit/85940eccef29ab3d52e08b1add3eb61d506d24e3))


### Bug Fixes

* context not being passed to reboot-manager ([85a6fd9](https://github.com/soerenschneider/sc-agent/commit/85a6fd942b8ace624baf81f8d2e0ce31d8552f8b))
* **deps:** bump github.com/prometheus/client_golang ([d71cd73](https://github.com/soerenschneider/sc-agent/commit/d71cd73b9443b13b0931b25e87ea048ca874454d))

## [1.6.0](https://github.com/soerenschneider/sc-agent/compare/v1.5.2...v1.6.0) (2024-10-20)


### Features

* increase resilience by adding optional timeout for vault login ([7fbddba](https://github.com/soerenschneider/sc-agent/commit/7fbddba6fa47756939f398445a07eac298183eaa))

## [1.5.2](https://github.com/soerenschneider/sc-agent/compare/v1.5.1...v1.5.2) (2024-10-11)


### Bug Fixes

* detect ubuntu and raspbian as debian derivates ([be580a5](https://github.com/soerenschneider/sc-agent/commit/be580a50d1b4afe9e408eb69688540a53388c1d1))
* only build packages service by default if on a supported distribution ([a5226a0](https://github.com/soerenschneider/sc-agent/commit/a5226a086196cb53e3785187d8b5c26f59b9f13e))

## [1.5.1](https://github.com/soerenschneider/sc-agent/compare/v1.5.0...v1.5.1) (2024-10-11)


### Bug Fixes

* detect alma linux and rocky linux as RHEL derivate ([5025005](https://github.com/soerenschneider/sc-agent/commit/50250054cc6c585bca89b108d8ca6cc7a67b6775))

## [1.5.0](https://github.com/soerenschneider/sc-agent/compare/v1.4.0...v1.5.0) (2024-10-11)


### Features

* add checker for required reboots based on os ([927ae91](https://github.com/soerenschneider/sc-agent/commit/927ae91652eb0fc34f3f9a0c6d90a7047902b0bc))
* continuously check for available updates ([1f90f56](https://github.com/soerenschneider/sc-agent/commit/1f90f56db2832b82d31f6d6cb6addb197417cd63))
* initial support for gathering system info ([5d539d0](https://github.com/soerenschneider/sc-agent/commit/5d539d0598ef1bad81a1b6e76de8353d8e82bd1a))
* re-use storage backend to allow fine grained control over target file ([b3e2364](https://github.com/soerenschneider/sc-agent/commit/b3e23640d3498887deb5f23715308263a9cbe67b))
* use existing storage backends for secrets replication ([128b61f](https://github.com/soerenschneider/sc-agent/commit/128b61fce96eb1e14ff8bb88a01673363d2f3578))


### Bug Fixes

* **deps:** bump github.com/getkin/kin-openapi from 0.127.0 to 0.128.0 ([569c64c](https://github.com/soerenschneider/sc-agent/commit/569c64c7038291a2f9bb60c6595119af1aeae012))
* **deps:** bump github.com/prometheus/common from 0.59.1 to 0.60.0 ([2ec432e](https://github.com/soerenschneider/sc-agent/commit/2ec432e737ee562f06c849c343536e4cc67839ee))
* **deps:** bump golang from 1.23.1 to 1.23.2 ([2609da1](https://github.com/soerenschneider/sc-agent/commit/2609da18469d5f93b25b51eeff1738e9683f172a))
* **deps:** bump golang.org/x/crypto from 0.27.0 to 0.28.0 ([4206e4f](https://github.com/soerenschneider/sc-agent/commit/4206e4fb05356a9d18daa2f04bd4ff4579962905))
* **deps:** bump golang.org/x/term from 0.24.0 to 0.25.0 ([aa02d59](https://github.com/soerenschneider/sc-agent/commit/aa02d59dc099594932991de9f29fad31cfc836c5))
* prevent newly added destinations not being written to ([f6d1117](https://github.com/soerenschneider/sc-agent/commit/f6d11174fe0783d153e298c46b0d5dadd7bbec99))

## [1.4.0](https://github.com/soerenschneider/sc-agent/compare/v1.3.0...v1.4.0) (2024-10-04)


### Features

* also build releases for arm7 ([e445b87](https://github.com/soerenschneider/sc-agent/commit/e445b87a27398a822bf5479ab05db5a097d5d43b))

## [1.3.0](https://github.com/soerenschneider/sc-agent/compare/v1.2.0...v1.3.0) (2024-10-01)


### Features

* add further metrics for http replication ([5e8ef76](https://github.com/soerenschneider/sc-agent/commit/5e8ef76eab4a5a2e6906360141f2a2c242d89635))


### Bug Fixes

* fix metric help ([642eebb](https://github.com/soerenschneider/sc-agent/commit/642eebbd88e0e9b675d21619c071a84935d751e1))
* fix metric type ([e99a19f](https://github.com/soerenschneider/sc-agent/commit/e99a19f4c2498b9137f7f85f6c5bfe1341890238))
* write to all backends ([b5cb9b9](https://github.com/soerenschneider/sc-agent/commit/b5cb9b9151206e3c95fed5b3390e9450296802f5))

## [1.2.0](https://github.com/soerenschneider/sc-agent/compare/v1.1.0...v1.2.0) (2024-09-30)


### Features

* allow writing replicated http items to multiple storage backends ([64fe211](https://github.com/soerenschneider/sc-agent/commit/64fe211f899cfbe048c84898efaaf97c8e34fb74))


### Bug Fixes

* **deps:** bump github.com/go-playground/validator/v10 ([982c293](https://github.com/soerenschneider/sc-agent/commit/982c293e7640c6fb9f1dce2065e5312e214194af))
* **deps:** bump github.com/hashicorp/vault/api from 1.14.0 to 1.15.0 ([d2a52c0](https://github.com/soerenschneider/sc-agent/commit/d2a52c05ff6e69b8e4ef857e168543a605c594bb))
* **deps:** bump github.com/oapi-codegen/oapi-codegen/v2 ([df2a521](https://github.com/soerenschneider/sc-agent/commit/df2a5217cfc80b2038132be75ea0334638605b3b))
* **deps:** bump github.com/prometheus/client_golang ([2202269](https://github.com/soerenschneider/sc-agent/commit/22022699cce8684fd4095dcd2fc7cfa85053090f))
* omitted 'dive' tag lead to items not being validated ([7f6a6d0](https://github.com/soerenschneider/sc-agent/commit/7f6a6d0569f1d6dbd6e11b8c74d3f86d3b524997))
* remove 'required' tag from sha256sum ([e790257](https://github.com/soerenschneider/sc-agent/commit/e790257a7c523fee58cdc18ce4afc39b41ea4fa2))

## [1.1.0](https://github.com/soerenschneider/sc-agent/compare/v1.0.1...v1.1.0) (2024-09-11)


### Features

* add metrics for http replication errors ([f9375c3](https://github.com/soerenschneider/sc-agent/commit/f9375c3c614688e5f64917ce3ed1d965510fac03))


### Bug Fixes

* also write metric when reading existing certificate ([02f8128](https://github.com/soerenschneider/sc-agent/commit/02f8128e2ff74c578f5cba5428518e4e71c5a725))
* **deps:** bump github.com/prometheus/client_golang ([d5b11ed](https://github.com/soerenschneider/sc-agent/commit/d5b11ed952311cb54a388fb177aad9c2d8b9df0d))
* **deps:** bump github.com/prometheus/common from 0.57.0 to 0.59.1 ([9e8395e](https://github.com/soerenschneider/sc-agent/commit/9e8395ef65f7ccacd766f568a5634a6569220a3e))
* **deps:** bump golang from 1.23.0 to 1.23.1 ([69b627e](https://github.com/soerenschneider/sc-agent/commit/69b627e91f37db4fae3f3d572f28d0d162779f13))
* **deps:** bump golang.org/x/crypto from 0.26.0 to 0.27.0 ([1b0be87](https://github.com/soerenschneider/sc-agent/commit/1b0be8762733f59488cdd297e99d34d2719884a6))
* **deps:** bump golang.org/x/term from 0.23.0 to 0.24.0 ([8aec0e6](https://github.com/soerenschneider/sc-agent/commit/8aec0e665632b9ad4ed4027f705332637e8bf787))
* fix checks ([ae488ea](https://github.com/soerenschneider/sc-agent/commit/ae488ea90b05d2fca2006b9bd582964729717362))
* require cert, key and ca if tls_client_auth=true ([f3fbb59](https://github.com/soerenschneider/sc-agent/commit/f3fbb591d3fa3ea455bf0f52cb0a613bced5a61b))
* trim strings before hashing ([14327d0](https://github.com/soerenschneider/sc-agent/commit/14327d0a94b24688ec033f15cce468361dd214fd))
* write metrics in case of file is already existent ([1ce18e1](https://github.com/soerenschneider/sc-agent/commit/1ce18e16622c0ddff6b09f1560ad34ee02f38f30))

## [1.0.1](https://github.com/soerenschneider/sc-agent/compare/v1.0.0...v1.0.1) (2024-09-02)


### Bug Fixes

* **deps:** bump github.com/prometheus/client_golang ([ea70ab9](https://github.com/soerenschneider/sc-agent/commit/ea70ab95be093ebd646ba952a09342539f224fa1))
* **deps:** bump github.com/prometheus/common from 0.55.0 to 0.57.0 ([9db5770](https://github.com/soerenschneider/sc-agent/commit/9db5770c869915c96e0045a8039f1637bfc744ca))
* set tls min version to tls1.3 ([7b57d3e](https://github.com/soerenschneider/sc-agent/commit/7b57d3eeb79a6ecc9bcd7baf39faf74647b8cba2))

## 1.0.0 (2024-08-23)


### Features

* dynamically load tls certificates ([700c624](https://github.com/soerenschneider/sc-agent/commit/700c6247afc37693e573067c9e284546f33959e9))


### Bug Fixes

* **deps:** bump github.com/prometheus/client_golang ([78815be](https://github.com/soerenschneider/sc-agent/commit/78815bec119f34128f9379cda89eef6201f95d1b))

## [1.1.0](https://github.com/soerenschneider/sc-agent/compare/v1.0.0...v1.1.0) (2024-08-05)


### Features

* add flag to print version and exit ([ba7f0d2](https://github.com/soerenschneider/sc-agent/commit/ba7f0d2b16a2b4c5b310e8ef18dd6f3be7f96280))
* support reading multiple config files from a config dir ([568f4b0](https://github.com/soerenschneider/sc-agent/commit/568f4b066f3db7b2f94ca8588f625f657fb9d9b9))
* watch releases on github and check if the currently agent is outdated ([1f9b665](https://github.com/soerenschneider/sc-agent/commit/1f9b66593f21efdb28b24ab7128f6303b0907157))

## 1.0.0 (2024-08-04)


### Bug Fixes

* **deps:** bump github.com/caarlos0/env/v11 from 11.1.0 to 11.2.0 ([0ec0a5c](https://github.com/soerenschneider/sc-agent/commit/0ec0a5cff9d6c5ce883f4cc7dfd88c1087259a68))
* **deps:** bump github.com/prometheus-community/pro-bing ([70ef057](https://github.com/soerenschneider/sc-agent/commit/70ef0578738620bb092c4158e355ed2ebfa1efa6))
* **deps:** bump github.com/prometheus/common from 0.54.0 to 0.55.0 ([af86d99](https://github.com/soerenschneider/sc-agent/commit/af86d9942a9e36c089f34c05b47cfa84be85957c))
* **deps:** bump github.com/rabbitmq/amqp091-go from 1.9.0 to 1.10.0 ([c909fd1](https://github.com/soerenschneider/sc-agent/commit/c909fd191845bc871af247bf125a6c02f7a5f41a))
* **deps:** bump golang.org/x/term from 0.20.0 to 0.22.0 ([d7b166d](https://github.com/soerenschneider/sc-agent/commit/d7b166d373ec9ee14191d1fa6039c7f23cec94f9))

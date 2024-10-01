# Changelog

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

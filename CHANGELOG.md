# Changelog

## [1.1.0](https://github.com/Hasras-code/PMT_WEB/compare/v1.0.0...v1.1.0) (2026-09-23)


### Features

* add empty hello world file ([8dc5c8d](https://github.com/Hasras-code/PMT_WEB/commit/8dc5c8d7bb8c64b45c75180164e025a2451ee8b5))

## 1.0.0 (2026-09-23)


### Features

* add HTTP Basic Auth for health endpoints and update configuration ([cd3d249](https://github.com/Hasras-code/PMT_WEB/commit/cd3d24906548555d1382027097aaa66cbcb5a4f1))
* add migration scripts for seeding platform admin role and related tests ([fabb218](https://github.com/Hasras-code/PMT_WEB/commit/fabb218b344d48a7e5f17b33b0dc28cabf33b1e0))
* add PublicHome page with hero section, achievements, events, gallery, and committee details; update email verification redirect ([ec910b6](https://github.com/Hasras-code/PMT_WEB/commit/ec910b61a4d98165eb8a675f40efeb255becd8b0))
* **api:** add OpenAPI documentation generation and routing ([7f42c8b](https://github.com/Hasras-code/PMT_WEB/commit/7f42c8b09212c13a5945854510e9be965b8a3299))
* **auth:** add combination field to user registration and update related logic ([6cca31e](https://github.com/Hasras-code/PMT_WEB/commit/6cca31e62621dbc644ec06cc08ebc42fb05761b4))
* **deploy:** add make-env.sh to generate .env.vps secrets on VPS ([a99c2df](https://github.com/Hasras-code/PMT_WEB/commit/a99c2df01692082285c04b5de8e4d4334a618342))
* **deploy:** add temporary IP-only VPS bundle in deploy/vps-temp ([3c19c16](https://github.com/Hasras-code/PMT_WEB/commit/3c19c16f5bf0238646eb068162a0d39288352b0a))
* **deploy:** add TUNNEL_URL share-link mode to vps-temp overlay ([3684d79](https://github.com/Hasras-code/PMT_WEB/commit/3684d79633cd95a24655d70f120af3c59aee0298))
* **deploy:** auto-deploy VPS on main via restricted deploy key ([479c3e5](https://github.com/Hasras-code/PMT_WEB/commit/479c3e5d4397ec7072d513cf15d93ea9e00588e6))
* **deploy:** remap temp VPS API to host port 8090 for shared box ([1d18368](https://github.com/Hasras-code/PMT_WEB/commit/1d18368cf7b93470801780e15f904e93d88af49e))
* enhance monitoring and profile features with role-based access control ([48c2e11](https://github.com/Hasras-code/PMT_WEB/commit/48c2e11e99259c20e87ef397baf8f4c23fb3979f))
* enhance registration process with batch ID and role assignments ([d3cd575](https://github.com/Hasras-code/PMT_WEB/commit/d3cd57532dfed9328b4f9d3203b4c5be891fca7a))
* **frontend:** add Figma-style React portal with admin and student views ([5d1f238](https://github.com/Hasras-code/PMT_WEB/commit/5d1f23878cb346b0d4868dad153594ad2c58ba65))
* **frontend:** support combination field and basic-auth health checks ([d03c852](https://github.com/Hasras-code/PMT_WEB/commit/d03c852ebc9971953526b58d45ba201be54ee01a))
* **frontend:** update footer message in Login component ([a0994f1](https://github.com/Hasras-code/PMT_WEB/commit/a0994f18fd166b0b4c5fd93e113a9b0f0bee84b0))
* **frontend:** update welcome message and description in Login component ([4fa3c9d](https://github.com/Hasras-code/PMT_WEB/commit/4fa3c9dcc1fb25fd520455250cf53b8ab917e5da))
* **frontend:** world-standard phase 0 — theme-safe correctness ([86c3e09](https://github.com/Hasras-code/PMT_WEB/commit/86c3e094924a4a70a1ef7f82ba105e88f9856490))
* **frontend:** world-standard phase 1 — app shell ([7eb5d63](https://github.com/Hasras-code/PMT_WEB/commit/7eb5d63bb80738398e20567d7e23a834f9925b8a))
* **frontend:** world-standard phase 2 — accessibility ([d068f15](https://github.com/Hasras-code/PMT_WEB/commit/d068f15723622c6598a2ca8104d5dc478d77797a))
* **frontend:** world-standard phase 2a — accessible edit dialog ([d0fa8cc](https://github.com/Hasras-code/PMT_WEB/commit/d0fa8cc5181fdc841151834e1e2f0c2a8723b6b1))
* **frontend:** world-standard phase 3 — loading, errors, guards ([0664563](https://github.com/Hasras-code/PMT_WEB/commit/0664563bac8e3d1afbeecc90bf5e22bb7566ff86))
* Implement fund transfer functionality ([8cb8230](https://github.com/Hasras-code/PMT_WEB/commit/8cb8230d328522ff22f85e8cb986d3f4610651f4))
* Implement Kuppi service for managing recorded lessons ([11cdf0a](https://github.com/Hasras-code/PMT_WEB/commit/11cdf0a61ae85836aa807f10c0f5d3b6b0879eef))
* update color scheme for dashboard tiles and login page messaging ([acad3e5](https://github.com/Hasras-code/PMT_WEB/commit/acad3e50e201318aea422d0edb63c8c260900895))
* update dependencies and improve user session management ([ffab3e8](https://github.com/Hasras-code/PMT_WEB/commit/ffab3e8fdc1d59c63a45966caec14ae3e52d89bd))
* update navigation to portal and enhance login redirection logic ([da2c20c](https://github.com/Hasras-code/PMT_WEB/commit/da2c20c71fa74304f5d37c2729eff590e4ff2287))
* update welcome message in Login component ([5ca45ab](https://github.com/Hasras-code/PMT_WEB/commit/5ca45ab328c4a2465e7138e9cc5c2164a8737e46))


### Bug Fixes

* **ci:** remove unused tokenInput type and legacySupportRoutes ([d121a56](https://github.com/Hasras-code/PMT_WEB/commit/d121a5652cae10115a1d639f64e00c25866a1d29))
* **ci:** use default GITHUB_TOKEN for release-please ([81ac011](https://github.com/Hasras-code/PMT_WEB/commit/81ac01157f298f4256acf7496a9c4f3d58754a22))
* **deploy:** mark vps-temp scripts executable ([cceab93](https://github.com/Hasras-code/PMT_WEB/commit/cceab93a0b261de0d19cf8f953496f9c6c60af50))
* **deploy:** scope container and volume lookups to PMT project only ([dde458a](https://github.com/Hasras-code/PMT_WEB/commit/dde458ab56dd586cd645c34fefe2a725ffca06bf))
* **deploy:** tolerate missing crontab for fresh deploy users ([bbfb055](https://github.com/Hasras-code/PMT_WEB/commit/bbfb055167db91b92a4c08e8a6feb32f66414f28))
* **deploy:** validate secrets and key format before VPS ssh ([cf9e0e3](https://github.com/Hasras-code/PMT_WEB/commit/cf9e0e3f8afb10faaf5278a90c6fd97f4a1dabba))
* **frontend:** resolve rebase conflicts with Funds feature ([cf1f4ef](https://github.com/Hasras-code/PMT_WEB/commit/cf1f4efdce8632c892d50e143574669d2b9c027c))
* Improve error handling in AuthorizeChecked function ([6a5740f](https://github.com/Hasras-code/PMT_WEB/commit/6a5740f31ccde49677cd9813779913257ef75588))


### Reverts

* **deploy:** remove live GitHub auto-deploy connection ([7dcc020](https://github.com/Hasras-code/PMT_WEB/commit/7dcc0208bb7f0eaadac69b6ec6dcfbc6cd7d928f))

# Contributing

## Signed commits are required

> [!IMPORTANT]
> All commits must be [signed](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-commits) (GPG, SSH, or S/MIME) to be merged into this repository. Pull requests with unsigned commits will need to be re-committed with signatures before they can be merged.

Thank you for your interest in contributing to the Loki data source for Grafana! We welcome contributions from the community.

Feel free to [browse open issues](https://github.com/grafana/grafana-loki-datasource/issues) or open a new one. For more general guidance, see [Grafana's Contributing Guide](https://github.com/grafana/grafana/blob/main/CONTRIBUTING.md).

This project adheres to the [Grafana Code of Conduct](https://github.com/grafana/grafana/blob/main/CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Prerequisites

- [Git](https://git-scm.com/)
- [Go](https://golang.org/dl/) (see [go.mod](go.mod) for the minimum required version)
- [Mage](https://magefile.org/)
- [Node.js LTS](https://nodejs.org)
- [npm](https://docs.npmjs.com/downloading-and-installing-node-js-and-npm) (see [package.json](package.json) for the minimum required version)
- [Docker](https://docs.docker.com/get-docker/)

## Frontend

1. Install dependencies:

   ```shell
   npm install
   ```

2. Build plugin in development mode and watch for changes:

   ```shell
   npm run dev
   ```

3. Build plugin in production mode:

   ```shell
   npm run build
   ```

4. Run frontend tests:

   ```shell
   npm run test:ci
   ```

## Backend

1. Build the backend binaries:

   ```shell
   mage -v
   ```

1. Run the backend tests, including the schema conformance tests:

   ```shell
   go test ./pkg/...
   ```

## Data source configuration schema

`pkg/schema/dsconfig.json` is the single source of truth for the data source configuration (`root`, `jsonData` and `secureJsonData`). Provisioning tools, docs and automation read it. See the [dsconfig README](https://github.com/grafana/dsconfig/tree/main/dsconfig), the [property reference](https://github.com/grafana/dsconfig/blob/main/dsconfig/schema.md) and the [JSON Schema](https://github.com/grafana/dsconfig/blob/main/dsconfig/schema.json).

The `$schema` URL is pinned to a dsconfig tag. Keep it equal to the `github.com/grafana/dsconfig/dsconfig` version in `go.mod`, and bump them together.

### Layout

| File | Purpose |
| --- | --- |
| `pkg/schema/dsconfig.json` | The schema. Edit this. |
| `pkg/schema/dsconfig_test.go` | Conformance test wiring and `SecureKeys`. |
| `pkg/schema/models/settings.go` | Test-only model of the plugin's `jsonData` keys. The backend does not read it. |
| `pkg/schema/*.gen.json` | Generated. Never hand-edit. The build copies them into `dist/schema/`. |

### Adding a new settings option

1. Declare the field in `dsconfig.json` under `fields` and add its id to `groups[].fieldRefs`. Ids follow `<target>_<key>`.
2. Add a struct field with a json tag equal to the schema `key` in `pkg/schema/models/settings.go`. Secrets have no struct field; add the key to `SecureKeys` in `dsconfig_test.go`.
3. Run `go generate ./pkg/schema/...` and commit the regenerated files.
4. Run `go test ./pkg/schema/...`.

### When the conformance suite fails

- `SchemaArtifactInSync`: the `*.gen.json` files are stale. Run `go generate ./pkg/schema/...`.
- `JSONDataMatchesStruct` / `JSONDataTypesMatchStruct`: a schema `jsonData` key and the struct json tags differ, or their types do. Fix the schema or the struct.
- `SecureValuesMatchLoadSettings`: `SecureKeys` and the schema `secureJsonData` keys differ. Update `SecureKeys`.

## Local development environment

`npm run server` starts a local Loki instance and a Grafana instance with the plugin pre-provisioned:

```shell
npm run server
```

You will need a running Loki instance. See the [Loki documentation](https://grafana.com/docs/loki/latest/get-started/) for setup instructions.

## E2E tests

```shell
npm run server
npm run e2e
```

Or, to install Playwright browsers first:

```shell
npx playwright install --with-deps
npm run server
npm run e2e
```

## Release

You need commit access to the repository to publish a release.

1. Update the version number in `package.json`.
2. Update `CHANGELOG.md` with the changes included in the release.
3. Open a PR with the changes and merge it.
4. Follow the release process described [here](https://enghub.grafana-ops.net/docs/default/component/grafana-plugins-platform/plugins-ci-github-actions/010-plugins-ci-github-actions/#cd_1).

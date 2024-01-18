# socket-dev-exporter

A simple Prometheus exporter to expose [Socket.dev](https://socket.dev/) scores for the latest versions of `@celo` NPM packages written in go.

This exporter exposes metrics in port `9101`, path `/metrics`, with the following format:

```txt
# HELP socket_score Shows socket.dev packages scores
# TYPE socket_score gauge
socket_score{package="@celo/0x-contracts",score="license",version="2.1.2-0.0"} 0.8629757195290285
socket_score{package="@celo/0x-contracts",score="maintenance",version="2.1.2-0.0"} 0.6968453019359488
socket_score{package="@celo/0x-contracts",score="miscellaneous",version="2.1.2-0.0"} 0
socket_score{package="@celo/0x-contracts",score="quality",version="2.1.2-0.0"} 0.6410426253533731
socket_score{package="@celo/0x-contracts",score="supplychainrisk",version="2.1.2-0.0"} 0.39592272547306173
socket_score{package="@celo/0x-contracts",score="vulnerability",version="2.1.2-0.0"} 0.25
...
```

## Configuration

3 environmental variables are available to configure this exporter:

- `API_TOKEN` (REQUIRED): A [Socket.dev](https://socket.dev/) API token.
- `LOG_LEVEL`: The [Logrus](https://github.com/sirupsen/logrus) log level. If not set, defaults to `info`.
- `PERIOD`: The period to refresh the [Socket.dev](https://socket.dev/) scores, in hours. If not set, defaults to `24`.

## Tests

Tests can be found in [`main_tests.go`](./main_test.go).

## CI/CD

The CI/CD pipeline is defined as [GitHub Action workflow](.github/workflows/ci-cd.yaml) with the following jobs:

- With each PR, commit to `main` or release the code will be built and tested.
- With each PR, a Docker image will be pushed to `us-west1-docker.pkg.dev/devopsre/dev-images/socket-exporter` with tag `test`.
- With each commit to `main`, a Docker image will be pushed to `us-west1-docker.pkg.dev/devopsre/socket-exporter/socket-exporter` with tag `latest`.
- With each release, a Docker image will be pushed to `us-west1-docker.pkg.dev/devopsre/socket-exporter/socket-exporter` with the same tag as the release tag.

The Dockerfile for building the Docker image can be found [here](./Dockerfile).

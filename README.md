# GitLab Orchard Executor

Custom [GitLab Runner](https://docs.gitlab.com/runner/) executor that runs CI jobs inside ephemeral macOS VMs managed by [Orchard](https://tart.run/orchard/quick-start/).

The executor itself runs on **Linux**. Orchard schedules the Tart VMs onto Apple Silicon workers.

Inspired by [cirruslabs/gitlab-tart-executor](https://github.com/cirruslabs/gitlab-tart-executor), but talks to an Orchard controller instead of running Tart locally.

## Why a capacity gate?

Orchard accepts `POST /v1/vms` even when no worker has free slots. Those VMs stay `pending` and pile up. This executor **waits for a free slot before creating a VM**, using worker resources (`org.cirruslabs.tart-vms` by default) and optional `--max-concurrent-vms`.

## Install

### Binary (Linux)

Download a release asset, or build:

```bash
go build -o gitlab-orchard-executor ./cmd/gitlab-orchard-executor
```

### Docker

```bash
docker pull ghcr.io/robinjanke/gitlab-orchard-executor:latest
```

## GitLab Runner configuration

Install [GitLab Runner](https://docs.gitlab.com/runner/install/) on a Linux host that can reach your Orchard controller, then configure a custom executor:

```toml
concurrent = 2

[[runners]]
  name = "application-platform-macos"
  url = "https://gitlab.example.com/"
  token = "GLRT_..."
  executor = "custom"
  builds_dir = "/Users/admin/builds"
  cache_dir = "/Users/admin/cache"
  [runners.feature_flags]
    FF_RESOLVE_FULL_TLS_CHAIN = false
  [runners.custom]
    config_exec = "/usr/local/bin/gitlab-orchard-executor"
    config_args = ["config", "--builds-dir", "/Users/admin/builds", "--cache-dir", "/Users/admin/cache"]
    prepare_exec = "/usr/local/bin/gitlab-orchard-executor"
    prepare_args = [
      "prepare",
      "--orchard-url", "https://orchard.example.com",
      "--orchard-service-account-name", "gitlab-runner",
      "--orchard-service-account-token", "REDACTED",
      "--cpu", "4",
      "--memory", "8192",
      "--max-concurrent-vms", "2",
      "--capacity-wait-timeout", "10m",
      "--image-pull-policy", "IfNotPresent"
    ]
    prepare_exec_timeout = 1200
    run_exec = "/usr/local/bin/gitlab-orchard-executor"
    run_args = ["run"]
    cleanup_exec = "/usr/local/bin/gitlab-orchard-executor"
    cleanup_args = ["cleanup"]
```

Credentials can also come from the environment (`ORCHARD_URL`, `ORCHARD_SERVICE_ACCOUNT_NAME`, `ORCHARD_SERVICE_ACCOUNT_TOKEN`) so tokens stay out of `config.toml`.

Set GitLab Runner `concurrent` to the number of VM slots you intend to use. The capacity gate is an additional safety net.

## Job configuration

```yaml
variables:
  MAC_RUNNER_TAG: "application-platform-macos"

build-ios:
  tags:
    - $MAC_RUNNER_TAG
  image: 10.0.0.109:9000/internal/modules/tart-images/gitlab-runner-macos/gitlab-runner-macos:v0.0.238
  script:
    - uname -a
```

The job `image:` becomes the Orchard VM image.

## Configuration reference

### `prepare` flags

| Flag | Default | Description |
| --- | --- | --- |
| `--orchard-url` | `$ORCHARD_URL` | Orchard controller base URL |
| `--orchard-service-account-name` | `$ORCHARD_SERVICE_ACCOUNT_NAME` | Basic-auth username |
| `--orchard-service-account-token` | `$ORCHARD_SERVICE_ACCOUNT_TOKEN` | Basic-auth password/token |
| `--orchard-trusted-certificate` | `$ORCHARD_TRUSTED_CERTIFICATE` | PEM of the Orchard controller cert (required for self-signed TLS) |
| `--cpu` | unset | VM CPU count |
| `--memory` | unset | VM memory in MiB |
| `--disk-size` | unset | VM disk size in GB |
| `--label key=value` | none | Restrict scheduling to workers that have these labels |
| `--resource key=value` | `org.cirruslabs.tart-vms=1` | Orchard resources requested by the VM |
| `--image-pull-policy` | `IfNotPresent` | `IfNotPresent` or `Always` |
| `--default-image` | unset | Fallback when the job has no `image:` |
| `--max-concurrent-vms` | `0` (unlimited) | Hard cap on `gitlab-*` VMs managed by this executor |
| `--capacity-wait-timeout` | `10m` | How long to wait for a free slot before failing prepare |
| `--vm-ready-timeout` | `15m` | How long to wait for the VM to reach `running` |
| `--allow-image` | none | Doublestar allow-list (repeatable) |
| `--ssh-username` / `--ssh-password` | `admin` / `admin` | Guest SSH credentials |
| `--headless` | `true` | Headless Tart VM |
| `--nested` | `false` | Nested virtualization |

### Targeting nodes (workers)

Orchard does not take a raw “node name” on create. Restrict placement with **labels**:

```bash
# Worker (once):
orchard worker run --labels model=macstudio,role=ci

# Executor prepare_args:
"--label", "model=macstudio",
"--label", "role=ci"
```

Resources are finite and accounted by the scheduler (unlike labels):

```bash
"--resource", "org.cirruslabs.tart-vms=1",
"--resource", "bandwidth-mbps=1000"
```

### Environment variables

| Name | Description |
| --- | --- |
| `ORCHARD_URL` | Controller URL |
| `ORCHARD_SERVICE_ACCOUNT_NAME` | Service account name |
| `ORCHARD_SERVICE_ACCOUNT_TOKEN` | Service account token |
| `ORCHARD_EXECUTOR_CPU` / `CUSTOM_ENV_ORCHARD_EXECUTOR_CPU` | Per-job CPU override |
| `ORCHARD_EXECUTOR_MEMORY` / `CUSTOM_ENV_ORCHARD_EXECUTOR_MEMORY` | Per-job memory (MiB) |
| `ORCHARD_EXECUTOR_LABELS` | Comma-separated `key=value` labels |
| `ORCHARD_EXECUTOR_RESOURCES` | Comma-separated `key=value` resources |
| `ORCHARD_EXECUTOR_IMAGE_PULL_POLICY` | `IfNotPresent` / `Always` |
| `ORCHARD_EXECUTOR_MAX_CONCURRENT_VMS` | Hard VM cap |
| `ORCHARD_EXECUTOR_SSH_USERNAME` / `ORCHARD_EXECUTOR_SSH_PASSWORD` | Guest SSH |
| `ORCHARD_EXECUTOR_SHELL` | Shell for `run` (e.g. `bash -l`) |

## Stages

1. **config** – report guest `builds_dir` / `cache_dir`
2. **prepare** – capacity gate → create VM → wait `running` → SSH probe
3. **run** – execute GitLab scripts over SSH (Orchard port-forward)
4. **cleanup** – delete the Orchard VM

VM names are deterministic: `gitlab-<CI_JOB_ID>`.

## License

MIT

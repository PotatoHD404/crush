# Locked-down Crush build

This fork of Crush is locked to a **single custom OpenAI-compatible provider** and makes
**no other network requests**. All external catalogs/telemetry/update/auth egress is
removed or blocked:

- The provider catalog services (**catwalk** `catwalk.charm.land`, **hyper**
  `hyper.charm.land`) are never contacted — `config.Providers()` returns only the custom
  provider baked in at build time (`internal/config/custom_provider.go`).
- A global **network guard** (`internal/netguard`) replaces the default HTTP transport at
  startup so the only reachable hosts are the model endpoint, an optional analytics
  endpoint, and loopback. Update checks, OAuth (Copilot/Hyper), PostHog telemetry
  (`data.charm.land`), and the agent web tools (DuckDuckGo/Sourcegraph/fetch/download) are
  all blocked because they use the default transport.
- Telemetry (`internal/event`) is **off by default**; it only initializes when an
  analytics endpoint is baked in, and then sends to *that* endpoint — never charm.land.

## Build-time configuration (Go linker flags)

The endpoint ("api route"), model identity, and analytics endpoint are baked in via
`-ldflags -X`. The API **key value** is never baked in — only the **name** of the env var
that holds it.

| ldflag var (`internal/config.<var>`) | Default                    | Purpose                              |
| ------------------------------------ | -------------------------- | ------------------------------------ |
| `customBaseURL`                      | `http://localhost:8080/v1` | OpenAI-compatible API route          |
| `customModelID`                      | `default`                  | Model id sent to the endpoint        |
| `customModelName`                    | = model id                 | Display name                         |
| `customProviderID`                   | `custom`                   | Internal provider id                 |
| `customProviderName`                 | `Custom`                   | Provider display name                |
| `customAPIKeyEnv`                    | `CRUSH_API_KEY`            | **Name** of the runtime key env var  |
| `customContextWindow`                | `128000`                   | Context window                       |
| `customMaxTokens`                    | `8192`                     | Max output tokens                    |
| `AnalyticsURL`                       | `` (empty → off)           | Optional analytics endpoint          |

Example build:

```bash
go build -ldflags "\
  -X 'github.com/charmbracelet/crush/internal/config.customBaseURL=https://gateway.example.com/v1' \
  -X 'github.com/charmbracelet/crush/internal/config.customModelID=my-model' \
  -X 'github.com/charmbracelet/crush/internal/config.customModelName=My Model' \
  -X 'github.com/charmbracelet/crush/internal/config.customAPIKeyEnv=MY_GATEWAY_KEY' \
  -X 'github.com/charmbracelet/crush/internal/config.AnalyticsURL=https://analytics.example.com'" \
  -o crush .
```

## Runtime (the key)

Only the *name* of the key env var is baked in; set its value at runtime:

```bash
export CRUSH_API_KEY="sk-..."   # or whatever customAPIKeyEnv was set to
./crush
```

Keyless local gateways (vLLM, Ollama, LM Studio, LiteLLM) work without the variable set
(the provider falls back to a `noauth` placeholder, which such gateways ignore).

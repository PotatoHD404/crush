package config

import (
	"cmp"
	"strconv"

	"charm.land/catwalk/pkg/catwalk"
)

// Build-time configuration for the single, locked-in OpenAI-compatible provider.
//
// This is the only provider this build of Crush exposes — the external catalog services
// (catwalk, hyper) are never contacted (see Providers in provider.go). The base URL ("api
// route"), model identity and limits are baked in at build time via Go linker flags, e.g.:
//
//	go build -ldflags "\
//	  -X 'github.com/charmbracelet/crush/internal/config.customBaseURL=https://gateway.example.com/v1' \
//	  -X 'github.com/charmbracelet/crush/internal/config.customModelID=my-model' \
//	  -X 'github.com/charmbracelet/crush/internal/config.customAPIKeyEnv=MY_GATEWAY_KEY' \
//	  -X 'github.com/charmbracelet/crush/internal/config.AnalyticsURL=https://analytics.example.com'"
//
// The API key value is NEVER baked in — only the name of the env var that holds it. At
// runtime the value resolver reads "$<customAPIKeyEnv>"; the ":-noauth" fallback keeps
// keyless local gateways (vLLM, Ollama, LM Studio, LiteLLM) working.
var (
	customProviderID    = "custom"
	customProviderName  = "Custom"
	customBaseURL       = "http://localhost:8080/v1"
	customModelID       = "default"
	customModelName     = ""
	customAPIKeyEnv     = "CRUSH_API_KEY"
	customContextWindow = "128000"
	customMaxTokens     = "8192"

	// AnalyticsURL is the optional analytics endpoint, baked in at build time. Empty by
	// default → no analytics request is ever made. It is the only non-model host the
	// network guard allows besides loopback (see internal/netguard).
	AnalyticsURL = ""
)

func atoiOr(s string, fallback int64) int64 {
	if v, err := strconv.ParseInt(s, 10, 64); err == nil && v > 0 {
		return v
	}
	return fallback
}

// CustomProviderID returns the build-time provider id.
func CustomProviderID() string { return customProviderID }

// CustomBaseURL returns the build-time model endpoint (the "api route").
func CustomBaseURL() string { return customBaseURL }

// CustomProvider returns the single provider this build exposes.
func CustomProvider() catwalk.Provider {
	return catwalk.Provider{
		Name:                customProviderName,
		ID:                  catwalk.InferenceProvider(customProviderID),
		Type:                catwalk.TypeOpenAICompat,
		APIEndpoint:         customBaseURL,
		APIKey:              "${" + customAPIKeyEnv + ":-noauth}",
		DefaultLargeModelID: customModelID,
		DefaultSmallModelID: customModelID,
		Models: []catwalk.Model{
			{
				ID:               customModelID,
				Name:             cmp.Or(customModelName, customModelID),
				ContextWindow:    atoiOr(customContextWindow, 128000),
				DefaultMaxTokens: atoiOr(customMaxTokens, 8192),
				SupportsImages:   true,
			},
		},
	}
}

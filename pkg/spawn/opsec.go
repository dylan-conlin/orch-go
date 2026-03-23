package spawn

import (
	"fmt"
	"net/http"
	"time"
)

// CheckOpsecProxy verifies the local OPSEC proxy is running and responsive.
// Returns nil if opsec is not required (enabled=false) or if proxy is healthy.
// Returns error if proxy is required but not running — this is a hard-blocking check.
func CheckOpsecProxy(enabled bool, port int) error {
	if !enabled {
		return nil
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
	if err != nil {
		return fmt.Errorf("opsec-proxy not running on :%d — run 'orch opsec start': %w", port, err)
	}
	defer resp.Body.Close()

	// tinyproxy returns various status codes for non-proxy requests (400, 403, etc.)
	// Any response means the proxy is alive.
	return nil
}

// OpsecEnvPrefix returns the shell export prefix for OPSEC environment variables.
// When enabled, exports OPSEC_SANDBOX=1 and proxy env vars.
// Returns empty string when disabled.
func OpsecEnvPrefix(enabled bool, port int) string {
	if !enabled {
		return ""
	}
	proxy := fmt.Sprintf("http://127.0.0.1:%d", port)
	return fmt.Sprintf(
		"export OPSEC_SANDBOX=1; export HTTP_PROXY=%s; export HTTPS_PROXY=%s; export ALL_PROXY=%s; ",
		proxy, proxy, proxy,
	)
}

// DefaultOpsecPort is the default port for the local OPSEC proxy.
const DefaultOpsecPort = 8199

// OpsecSettingsPath returns the path to the OPSEC worker settings.json.
const OpsecSettingsPath = "~/.orch/opsec/worker-settings.json"

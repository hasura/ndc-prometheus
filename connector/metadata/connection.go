package metadata

import (
	"encoding/json"
	"fmt"

	"github.com/prometheus/common/config"
)

// ClientSettings contain information for the Prometheus server that the client connects to
type ClientSettings struct {
	// The endpoint of the Prometheus server.
	URL EnvironmentValue `json:"url" yaml:"url"`
	// The maximum amount of seconds a dial will wait for a connect to complete. The default is no timeout.
	Timeout int `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	// Specifies the interval in seconds between keep-alive probes for an active network connection. The default is 15 seconds.
	KeepAlive int `json:"keep_alive,omitempty" yaml:"keep_alive,omitempty"`

	// The HTTP basic authentication credentials for the targets.
	BasicAuth *config.BasicAuth `yaml:"basic_auth,omitempty" json:"basic_auth,omitempty"`
	// The HTTP authorization credentials for the targets.
	Authorization *config.Authorization `yaml:"authorization,omitempty" json:"authorization,omitempty"`
	// The OAuth2 client credentials used to fetch a token for the targets.
	OAuth2 *config.OAuth2 `yaml:"oauth2,omitempty" json:"oauth2,omitempty"`
	// TLSConfig to use to connect to the targets.
	TLSConfig config.TLSConfig `yaml:"tls_config,omitempty" json:"tls_config,omitempty"`
	// FollowRedirects specifies whether the client should follow HTTP 3xx redirects.
	// The omitempty flag is not set, because it would be hidden from the
	// marshalled configuration when set to false.
	FollowRedirects bool `yaml:"follow_redirects,omitempty" json:"follow_redirects,omitempty"`
	// EnableHTTP2 specifies whether the client should configure HTTP2.
	// The omitempty flag is not set, because it would be hidden from the
	// marshalled configuration when set to false.
	EnableHTTP2 bool `yaml:"enable_http2,omitempty" json:"enable_http2,omitempty"`
	// Proxy configuration.
	config.ProxyConfig `yaml:",inline"`
	// HTTPHeaders specify headers to inject in the requests. Those headers
	// could be marshalled back to the users.
	HTTPHeaders *config.Headers `yaml:"http_headers,omitempty" json:"http_headers,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (cs *ClientSettings) UnmarshalJSON(b []byte) error {
	type Plain ClientSettings
	var plain Plain
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}

	u, err := plain.URL.Get()
	if err != nil || u == "" {
		return fmt.Errorf("invalid client URL %s", err)
	}

	*cs = ClientSettings(plain)
	return nil
}

// ToHTTPClientConfig converts client settings to Prometheus client's HTTPClientConfig
func (cs ClientSettings) ToHTTPClientConfig() config.HTTPClientConfig {
	return config.HTTPClientConfig{
		BasicAuth:       cs.BasicAuth,
		Authorization:   cs.Authorization,
		OAuth2:          cs.OAuth2,
		TLSConfig:       cs.TLSConfig,
		FollowRedirects: cs.FollowRedirects,
		EnableHTTP2:     cs.EnableHTTP2,
		ProxyConfig:     cs.ProxyConfig,
		HTTPHeaders:     cs.HTTPHeaders,
	}
}

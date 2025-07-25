package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hasura/ndc-sdk-go/utils"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"google.golang.org/api/option"
	apihttp "google.golang.org/api/transport/http"
)

// ClientSettings contain information for the Prometheus server that the client connects to.
type ClientSettings struct {
	// Proxy configuration.
	*ProxyConfig `yaml:",inline"`

	// The endpoint of the Prometheus server.
	URL utils.EnvString `json:"url"                        yaml:"url"`
	// The authentication configuration
	Authentication *AuthConfig `json:"authentication,omitempty"   yaml:"authentication,omitempty"`
	// The default timeout in seconds for Prometheus requests. The default is no timeout.
	Timeout *model.Duration `json:"timeout,omitempty"          yaml:"timeout,omitempty"`
	// TLSConfig to use to connect to the targets.
	TLSConfig config.TLSConfig `json:"tls_config,omitempty"       yaml:"tls_config,omitempty"`
	// FollowRedirects specifies whether the client should follow HTTP 3xx redirects.
	// The omitempty flag is not set, because it would be hidden from the
	// marshalled configuration when set to false.
	FollowRedirects bool `json:"follow_redirects,omitempty" yaml:"follow_redirects,omitempty"`
	// EnableHTTP2 specifies whether the client should configure HTTP2.
	// The omitempty flag is not set, because it would be hidden from the
	// marshalled configuration when set to false.
	EnableHTTP2 bool `json:"enable_http2,omitempty"     yaml:"enable_http2,omitempty"`
	// HTTPHeaders specify headers to inject in the requests. Those headers
	// could be marshalled back to the users.
	HTTPHeaders http.Header `json:"http_headers,omitempty"     yaml:"http_headers,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (cs *ClientSettings) UnmarshalJSON(b []byte) error {
	type Plain ClientSettings

	var plain Plain

	if err := json.Unmarshal(b, &plain); err != nil { //nolint:musttag
		return err
	}

	u, err := plain.URL.Get()
	if err != nil || u == "" {
		return fmt.Errorf("invalid client URL %w", err)
	}

	*cs = ClientSettings(plain)

	return nil
}

// getHTTPClientConfig converts client settings to Prometheus client's HTTPClientConfig.
func (cs ClientSettings) getHTTPClientConfig() (*config.HTTPClientConfig, error) {
	result := &config.HTTPClientConfig{
		TLSConfig:       cs.TLSConfig,
		FollowRedirects: cs.FollowRedirects,
		EnableHTTP2:     cs.EnableHTTP2,
		HTTPHeaders:     cs.getHTTPHeaders(),
	}

	if cs.ProxyConfig != nil {
		pc, err := cs.toClientConfig()
		if err != nil {
			return nil, err
		}

		result.ProxyConfig = *pc
	}

	if cs.Authentication == nil {
		return result, nil
	}

	if cs.Authentication.Authorization != nil {
		au, err := cs.Authentication.Authorization.toClientConfig()
		if err != nil {
			return nil, err
		}

		result.Authorization = au
	}

	if cs.Authentication.OAuth2 != nil {
		au, err := cs.Authentication.OAuth2.toClientConfig()
		if err != nil {
			return nil, err
		}

		result.OAuth2 = au
	}

	if cs.Authentication.BasicAuth != nil {
		au, err := cs.Authentication.BasicAuth.toClientConfig()
		if err != nil {
			return nil, err
		}

		result.BasicAuth = au
	}

	return result, nil
}

func (cs ClientSettings) getHTTPHeaders() *config.Headers {
	result := config.Headers{
		Headers: make(map[string]config.Header),
	}

	for k, v := range cs.HTTPHeaders {
		result.Headers[k] = config.Header{
			Values: v,
		}
	}

	return &result
}

func (cs ClientSettings) createHttpClient(ctx context.Context) (*http.Client, error) {
	httpClient, err := cs.createGoogleHttpClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize the google http client: %w", err)
	}

	if httpClient != nil {
		return httpClient, nil
	}

	clientConfig, err := cs.getHTTPClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize the prometheus client config: %w", err)
	}

	return config.NewClientFromConfig(*clientConfig, "ndc-prometheus")
}

func (cs ClientSettings) createGoogleHttpClient(ctx context.Context) (*http.Client, error) {
	if cs.Authentication == nil || cs.Authentication.Google == nil {
		return nil, nil
	}

	opts := []option.ClientOption{
		option.WithScopes("https://www.googleapis.com/auth/monitoring.read"),
	}

	if cs.Authentication.Google.Credentials != nil {
		credJSON, err := cs.Authentication.Google.Credentials.Get()
		if err != nil {
			return nil, err
		}

		if credJSON != "" {
			if cs.Authentication.Google.Encoding != nil &&
				*cs.Authentication.Google.Encoding == CredentialsEncodingBase64 {
				credByte, err := base64.StdEncoding.DecodeString(credJSON)
				if err != nil {
					return nil, err
				}

				credJSON = string(credByte)
			}

			opts = append(opts, option.WithCredentialsJSON([]byte(credJSON)))
		}
	} else if cs.Authentication.Google.CredentialsFile != nil {
		credFile, err := cs.Authentication.Google.CredentialsFile.Get()
		if err != nil {
			return nil, err
		}

		if credFile != "" {
			opts = append(opts, option.WithCredentialsFile(credFile))
		}
	}

	rt, err := config.NewRoundTripperFromConfigWithContext(ctx, config.HTTPClientConfig{
		TLSConfig:       cs.TLSConfig,
		EnableHTTP2:     cs.EnableHTTP2,
		FollowRedirects: cs.FollowRedirects,
		HTTPHeaders:     cs.getHTTPHeaders(),
	}, "ndc-prometheus")
	if err != nil {
		return nil, err
	}

	transport, err := apihttp.NewTransport(ctx, rt, opts...)
	if err != nil {
		return nil, fmt.Errorf(
			"error occurred while fetching GCP transport while setting up client for prometheus: %w",
			err,
		)
	}

	return &http.Client{
		Transport: transport,
	}, nil
}

// AuthConfig the authentication configuration.
type AuthConfig struct {
	// The HTTP basic authentication credentials for the targets.
	BasicAuth *BasicAuthConfig `json:"basic,omitempty"         yaml:"basic,omitempty"`
	// The HTTP authorization credentials for the targets.
	Authorization *AuthorizationConfig `json:"authorization,omitempty" yaml:"authorization,omitempty"`
	// The OAuth2 client credentials used to fetch a token for the targets.
	OAuth2 *OAuth2Config `json:"oauth2,omitempty"        yaml:"oauth2,omitempty"`
	// The Google client credentials used to fetch a token for the targets.
	Google *GoogleAuthConfig `json:"google,omitempty"        yaml:"google,omitempty"`
}

// BasicAuth the HTTP basic authentication credentials for the targets.
type BasicAuthConfig struct {
	Username utils.EnvString `json:"username" yaml:"username"`
	Password utils.EnvString `json:"password" yaml:"password"`
}

func (bac BasicAuthConfig) toClientConfig() (*config.BasicAuth, error) {
	username, err := bac.Username.Get()
	if err != nil {
		return nil, fmt.Errorf("basic auth username: %w", err)
	}

	password, err := bac.Password.Get()
	if err != nil {
		return nil, fmt.Errorf("basic auth password: %w", err)
	}

	return &config.BasicAuth{
		Username: username,
		Password: config.Secret(password),
	}, nil
}

// AuthorizationConfig the HTTP authorization credentials for the targets.
type AuthorizationConfig struct {
	Type        utils.EnvString `json:"type"        yaml:"type"`
	Credentials utils.EnvString `json:"credentials" yaml:"credentials"`
}

func (hac AuthorizationConfig) toClientConfig() (*config.Authorization, error) {
	authType, err := hac.Type.Get()
	if err != nil {
		return nil, fmt.Errorf("authorization type: %w", err)
	}

	cred, err := hac.Credentials.Get()
	if err != nil {
		return nil, fmt.Errorf("authorization credentials: %w", err)
	}

	return &config.Authorization{
		Type:        authType,
		Credentials: config.Secret(cred),
	}, nil
}

// OAuth2Config the OAuth2 client credentials used to fetch a token for the targets.
type OAuth2Config struct {
	*ProxyConfig `yaml:",inline"`

	ClientID       utils.EnvString   `json:"client_id"                 yaml:"client_id"`
	ClientSecret   utils.EnvString   `json:"client_secret"             yaml:"client_secret"`
	TokenURL       utils.EnvString   `json:"token_url"                 yaml:"token_url"`
	Scopes         []string          `json:"scopes,omitempty"          yaml:"scopes,omitempty"`
	EndpointParams map[string]string `json:"endpoint_params,omitempty" yaml:"endpoint_params,omitempty"`
	TLSConfig      config.TLSConfig  `                                 yaml:"tls_config,omitempty"`
}

func (oc OAuth2Config) toClientConfig() (*config.OAuth2, error) {
	clientId, err := oc.ClientID.Get()
	if err != nil {
		return nil, fmt.Errorf("oauth2 client_id: %w", err)
	}

	clientSecret, err := oc.ClientSecret.Get()
	if err != nil {
		return nil, fmt.Errorf("oauth2 client_secret: %w", err)
	}

	tokenURL, err := oc.TokenURL.Get()
	if err != nil {
		return nil, fmt.Errorf("oauth2 token_url: %w", err)
	}

	result := &config.OAuth2{
		ClientID:       clientId,
		ClientSecret:   config.Secret(clientSecret),
		TokenURL:       tokenURL,
		Scopes:         oc.Scopes,
		EndpointParams: oc.EndpointParams,
		TLSConfig:      oc.TLSConfig,
	}

	if oc.ProxyConfig != nil {
		pc, err := oc.ProxyConfig.toClientConfig()
		if err != nil {
			return nil, err
		}

		result.ProxyConfig = *pc
	}

	return result, nil
}

// CredentialsEncoding the encoding of credentials string.
type CredentialsEncoding string

const (
	CredentialsEncodingPlainText CredentialsEncoding = "plaintext"
	CredentialsEncodingBase64    CredentialsEncoding = "base64"
)

// GoogleAuth the Google client credentials used to fetch a token for the targets.
type GoogleAuthConfig struct {
	Encoding *CredentialsEncoding `json:"encoding,omitempty"         jsonschema:"enum=plaintext,enum=base64,default=plaintext" yaml:"encoding,omitempty"`
	// Text of the Google credential JSON
	Credentials *utils.EnvString `json:"credentials,omitempty"                                                                yaml:"credentials,omitempty"`
	// Path of the Google credential file
	CredentialsFile *utils.EnvString `json:"credentials_file,omitempty"                                                           yaml:"credentials_file,omitempty"`
}

// ProxyConfig the proxy configuration.
type ProxyConfig struct {
	// HTTP proxy server to use to connect to the targets.
	ProxyURL string `json:"proxy_url,omitempty"              yaml:"proxy_url,omitempty"`
	// NoProxy contains addresses that should not use a proxy.
	NoProxy string `json:"no_proxy,omitempty"               yaml:"no_proxy,omitempty"`
	// ProxyFromEnvironment makes use of net/http ProxyFromEnvironment function
	// to determine proxies.
	ProxyFromEnvironment bool `json:"proxy_from_environment,omitempty" yaml:"proxy_from_environment,omitempty"`
	// ProxyConnectHeader optionally specifies headers to send to
	// proxies during CONNECT requests. Assume that at least _some_ of
	// these headers are going to contain secrets and use Secret as the
	// value type instead of string.
	ProxyConnectHeader config.ProxyHeader `json:"proxy_connect_header,omitempty"   yaml:"proxy_connect_header,omitempty"`
}

func (oc ProxyConfig) toClientConfig() (*config.ProxyConfig, error) {
	result := &config.ProxyConfig{
		NoProxy:              oc.NoProxy,
		ProxyFromEnvironment: oc.ProxyFromEnvironment,
		ProxyConnectHeader:   oc.ProxyConnectHeader,
	}

	if oc.ProxyURL != "" {
		u, err := url.Parse(oc.ProxyURL)
		if err != nil {
			return nil, err
		}

		result.ProxyURL = config.URL{URL: u}
	}

	return result, nil
}

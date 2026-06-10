package client

import (
	"context"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hasura/goenvconf"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"gotest.tools/v3/assert"
)

func createTestClient(t *testing.T) *Client {
	c, err := NewClient(context.TODO(), ClientSettings{
		URL: goenvconf.NewEnvStringValue("http://localhost:9090"),
		Authentication: &AuthConfig{
			BasicAuth: &BasicAuthConfig{
				Username: goenvconf.NewEnvStringValue("admin"),
				Password: goenvconf.NewEnvStringValue("test"),
			},
		},
	})
	assert.NilError(t, err)
	return c
}

func TestNewClient(t *testing.T) {
	gcpCred := `{
	"type": "service_account",
  "project_id": "some-test-account",
  "private_key_id": "some-key-id",
  "private_key": "-----BEGIN PRIVATE KEY-----\n-----END PRIVATE KEY-----\n",
  "client_email": "some-test@test-account.iam.gserviceaccount.com",
  "client_id": "01234567890",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/some-test@test-account.iam.gserviceaccount.com",
  "universe_domain": "googleapis.com"
}`

	gcpCredBase64 := base64.StdEncoding.EncodeToString([]byte(gcpCred))
	tmpDir := t.TempDir()
	gcpCredPath := filepath.Join(tmpDir, "service_account.json")
	assert.NilError(t, os.WriteFile(gcpCredPath, []byte(gcpCred), 0644))

	testCases := []struct {
		Name     string
		Config   ClientSettings
		ErrorMsg string
	}{
		{
			Name:     "empty_url",
			Config:   ClientSettings{},
			ErrorMsg: "invalid Prometheus URL: EmptyEnv: require either value or env",
		},
		{
			Name: "empty_url_2",
			Config: ClientSettings{
				URL: goenvconf.NewEnvStringValue(""),
			},
			ErrorMsg: errEndpointRequired.Error(),
		},
		{
			Name: "invalid_port",
			Config: ClientSettings{
				URL: goenvconf.NewEnvStringValue("http://localhost:abc"),
			},
			ErrorMsg: "invalid Prometheus URL: parse \"http://localhost:abc\": invalid port \":abc\" after host",
		},
		{
			Name: "no_auth",
			Config: ClientSettings{
				URL: goenvconf.NewEnvStringValue("http://localhost:9090"),
			},
		},
		{
			Name: "basic_auth_empty_username",
			Config: ClientSettings{
				URL: goenvconf.NewEnvStringValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					BasicAuth: &BasicAuthConfig{},
				},
			},
			ErrorMsg: "failed to initialize the prometheus client config: basic auth username: EmptyEnv: require either value or env",
		},
		{
			Name: "basic_auth_empty_password",
			Config: ClientSettings{
				URL: goenvconf.NewEnvStringValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					BasicAuth: &BasicAuthConfig{
						Username: goenvconf.NewEnvStringValue("admin"),
					},
				},
			},
			ErrorMsg: "failed to initialize the prometheus client config: basic auth password: EmptyEnv: require either value or env",
		},
		{
			Name: "http_auth",
			Config: ClientSettings{
				URL: goenvconf.NewEnvStringValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					Authorization: &AuthorizationConfig{
						Type:        goenvconf.NewEnvStringValue("Bearer"),
						Credentials: goenvconf.NewEnvStringValue("abc"),
					},
				},
				Timeout:     defaultClientOptions.timeout,
				EnableHTTP2: true,
				TLSConfig: config.TLSConfig{
					InsecureSkipVerify: true,
				},
				HTTPHeaders: http.Header{
					"foo": []string{"bar"},
				},
				ProxyConfig: &ProxyConfig{
					ProxyURL: "http://localhost:3000",
				},
			},
		},
		{
			Name: "http_auth_empty_type",
			Config: ClientSettings{
				URL: goenvconf.NewEnvStringValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					Authorization: &AuthorizationConfig{},
				},
			},
			ErrorMsg: "failed to initialize the prometheus client config: authorization type: EmptyEnv: require either value or env",
		},
		{
			Name: "http_auth_empty_credentials",
			Config: ClientSettings{
				URL: goenvconf.NewEnvStringValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					Authorization: &AuthorizationConfig{
						Type: goenvconf.NewEnvStringValue("Bearer"),
					},
				},
			},
			ErrorMsg: "failed to initialize the prometheus client config: authorization credentials: EmptyEnv: require either value or env",
		},
		{
			Name: "gcp_auth",
			Config: ClientSettings{
				URL: goenvconf.NewEnvStringValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					Google: &GoogleAuthConfig{
						Encoding:    new(CredentialsEncodingBase64),
						Credentials: new(goenvconf.NewEnvStringValue(gcpCredBase64)),
					},
				},
			},
		},
		{
			Name: "gcp_auth_file",
			Config: ClientSettings{
				URL: goenvconf.NewEnvStringValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					Google: &GoogleAuthConfig{
						CredentialsFile: new(goenvconf.NewEnvStringValue(gcpCredPath)),
					},
				},
			},
		},
		{
			Name: "oauth",
			Config: ClientSettings{
				URL: goenvconf.NewEnvStringValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					OAuth2: &OAuth2Config{
						ClientID:     goenvconf.NewEnvStringValue("client-id"),
						ClientSecret: goenvconf.NewEnvStringValue("client-secret"),
						TokenURL:     goenvconf.NewEnvStringValue("http://localhost:4444/oauth2/token"),
						ProxyConfig: &ProxyConfig{
							NoProxy: "test",
						},
					},
				},
			},
		},
		{
			Name: "oauth_empty",
			Config: ClientSettings{
				URL: goenvconf.NewEnvStringValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					OAuth2: &OAuth2Config{},
				},
			},
			ErrorMsg: "failed to initialize the prometheus client config: oauth2 client_id: EmptyEnv: require either value or env",
		},
		{
			Name: "oauth_client_secret_empty",
			Config: ClientSettings{
				URL: goenvconf.NewEnvStringValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					OAuth2: &OAuth2Config{
						ClientID: goenvconf.NewEnvStringValue("client-id"),
					},
				},
			},
			ErrorMsg: "failed to initialize the prometheus client config: oauth2 client_secret: EmptyEnv: require either value or env",
		},
		{
			Name: "oauth_token_url_empty",
			Config: ClientSettings{
				URL: goenvconf.NewEnvStringValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					OAuth2: &OAuth2Config{
						ClientID:     goenvconf.NewEnvStringValue("client-id"),
						ClientSecret: goenvconf.NewEnvStringValue("client-secret"),
					},
				},
			},
			ErrorMsg: "failed to initialize the prometheus client config: oauth2 token_url: EmptyEnv: require either value or env",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := NewClient(context.TODO(), tc.Config, WithTimeout(new(model.Duration(time.Minute))))
			if tc.ErrorMsg == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.ErrorMsg)
			}
		})
	}
}

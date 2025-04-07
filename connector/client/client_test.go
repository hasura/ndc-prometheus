package client

import (
	"context"
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hasura/ndc-prometheus/connector/types"
	"github.com/hasura/ndc-sdk-go/utils"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"gotest.tools/v3/assert"
)

func createTestClient(t *testing.T) *Client {
	c, err := NewClient(context.TODO(), ClientSettings{
		URL: types.NewEnvironmentValue("http://localhost:9090"),
		Authentication: &AuthConfig{
			BasicAuth: &BasicAuthConfig{
				Username: types.NewEnvironmentValue("admin"),
				Password: types.NewEnvironmentValue("test"),
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
	assert.NilError(t, os.WriteFile(gcpCredPath, []byte(gcpCred), 0o644))

	testCases := []struct {
		Name     string
		Config   ClientSettings
		ErrorMsg string
	}{
		{
			Name:     "empty_url",
			Config:   ClientSettings{},
			ErrorMsg: "invalid Prometheus URL: require either value or valueFromEnv",
		},
		{
			Name: "empty_url_2",
			Config: ClientSettings{
				URL: types.NewEnvironmentValue(""),
			},
			ErrorMsg: errEndpointRequired.Error(),
		},
		{
			Name: "invalid_port",
			Config: ClientSettings{
				URL: types.NewEnvironmentValue("http://localhost:abc"),
			},
			ErrorMsg: "invalid Prometheus URL: parse \"http://localhost:abc\": invalid port \":abc\" after host",
		},
		{
			Name: "no_auth",
			Config: ClientSettings{
				URL: types.NewEnvironmentValue("http://localhost:9090"),
			},
		},
		{
			Name: "basic_auth_empty_username",
			Config: ClientSettings{
				URL: types.NewEnvironmentValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					BasicAuth: &BasicAuthConfig{},
				},
			},
			ErrorMsg: "failed to initialize the prometheus client config: basic auth username: require either value or valueFromEnv",
		},
		{
			Name: "basic_auth_empty_password",
			Config: ClientSettings{
				URL: types.NewEnvironmentValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					BasicAuth: &BasicAuthConfig{
						Username: types.NewEnvironmentValue("admin"),
					},
				},
			},
			ErrorMsg: "failed to initialize the prometheus client config: basic auth password: require either value or valueFromEnv",
		},
		{
			Name: "http_auth",
			Config: ClientSettings{
				URL: types.NewEnvironmentValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					Authorization: &AuthorizationConfig{
						Type:        types.NewEnvironmentValue("Bearer"),
						Credentials: types.NewEnvironmentValue("abc"),
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
				URL: types.NewEnvironmentValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					Authorization: &AuthorizationConfig{},
				},
			},
			ErrorMsg: "failed to initialize the prometheus client config: authorization type: require either value or valueFromEnv",
		},
		{
			Name: "http_auth_empty_credentials",
			Config: ClientSettings{
				URL: types.NewEnvironmentValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					Authorization: &AuthorizationConfig{
						Type: types.NewEnvironmentValue("Bearer"),
					},
				},
			},
			ErrorMsg: "failed to initialize the prometheus client config: authorization credentials: require either value or valueFromEnv",
		},
		{
			Name: "gcp_auth",
			Config: ClientSettings{
				URL: types.NewEnvironmentValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					Google: &GoogleAuthConfig{
						Encoding:    utils.ToPtr(CredentialsEncodingBase64),
						Credentials: utils.ToPtr(types.NewEnvironmentValue(gcpCredBase64)),
					},
				},
			},
		},
		{
			Name: "gcp_auth_file",
			Config: ClientSettings{
				URL: types.NewEnvironmentValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					Google: &GoogleAuthConfig{
						CredentialsFile: utils.ToPtr(types.NewEnvironmentValue(gcpCredPath)),
					},
				},
			},
		},
		{
			Name: "oauth",
			Config: ClientSettings{
				URL: types.NewEnvironmentValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					OAuth2: &OAuth2Config{
						ClientID:     types.NewEnvironmentValue("client-id"),
						ClientSecret: types.NewEnvironmentValue("client-secret"),
						TokenURL: types.NewEnvironmentValue(
							"http://localhost:4444/oauth2/token",
						),
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
				URL: types.NewEnvironmentValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					OAuth2: &OAuth2Config{},
				},
			},
			ErrorMsg: "failed to initialize the prometheus client config: oauth2 client_id: require either value or valueFromEnv",
		},
		{
			Name: "oauth_client_secret_empty",
			Config: ClientSettings{
				URL: types.NewEnvironmentValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					OAuth2: &OAuth2Config{
						ClientID: types.NewEnvironmentValue("client-id"),
					},
				},
			},
			ErrorMsg: "failed to initialize the prometheus client config: oauth2 client_secret: require either value or valueFromEnv",
		},
		{
			Name: "oauth_token_url_empty",
			Config: ClientSettings{
				URL: types.NewEnvironmentValue("http://localhost:9090"),
				Authentication: &AuthConfig{
					OAuth2: &OAuth2Config{
						ClientID:     types.NewEnvironmentValue("client-id"),
						ClientSecret: types.NewEnvironmentValue("client-secret"),
					},
				},
			},
			ErrorMsg: "failed to initialize the prometheus client config: oauth2 token_url: require either value or valueFromEnv",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			_, err := NewClient(
				context.TODO(),
				tc.Config,
				WithTimeout(utils.ToPtr(model.Duration(time.Minute))),
				WithUnixTimeUnit(UnixTimeSecond),
			)
			if tc.ErrorMsg == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.ErrorMsg)
			}
		})
	}
}

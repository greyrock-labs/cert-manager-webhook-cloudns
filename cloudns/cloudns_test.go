package cloudns

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// lego v5 made its platform/tester helper internal, so the small amount of it
// this file used is reimplemented here on top of t.Setenv.

var providerEnvVars = []string{
	"CLOUDNS_AUTH_ID",
	"CLOUDNS_AUTH_ID_TYPE",
	"CLOUDNS_AUTH_PASSWORD",
}

const domainEnvVar = "CLOUDNS_DOMAIN"

// liveEnv is the environment as it stood before any test touched it. The live
// tests read it directly and are skipped unless every variable is populated.
var liveEnv = func() map[string]string {
	env := make(map[string]string, len(providerEnvVars)+1)
	for _, name := range append(providerEnvVars, domainEnvVar) {
		env[name] = os.Getenv(name)
	}
	return env
}()

func isLiveTest() bool {
	for _, name := range append(providerEnvVars, domainEnvVar) {
		if liveEnv[name] == "" {
			return false
		}
	}
	return true
}

// applyEnv clears every provider variable, then applies the given ones. An
// empty value is equivalent to unset as far as lego's env package is
// concerned, and t.Setenv restores the previous values when the test ends.
func applyEnv(t *testing.T, vars map[string]string) {
	t.Helper()
	for _, name := range providerEnvVars {
		t.Setenv(name, vars[name])
	}
}

func TestNewDNSProvider(t *testing.T) {
	testCases := []struct {
		desc     string
		envVars  map[string]string
		expected string
	}{
		{
			desc: "success default",
			envVars: map[string]string{
				"CLOUDNS_AUTH_ID":       "123",
				"CLOUDNS_AUTH_PASSWORD": "456",
			},
		},
		{
			desc: "success auth-id",
			envVars: map[string]string{
				"CLOUDNS_AUTH_ID":       "123",
				"CLOUDNS_AUTH_ID_TYPE":  "auth-id",
				"CLOUDNS_AUTH_PASSWORD": "456",
			},
		},
		{
			desc: "success sub-auth-id",
			envVars: map[string]string{
				"CLOUDNS_AUTH_ID":       "123",
				"CLOUDNS_AUTH_ID_TYPE":  "sub-auth-id",
				"CLOUDNS_AUTH_PASSWORD": "456",
			},
		},
		{
			desc: "missing credentials",
			envVars: map[string]string{
				"CLOUDNS_AUTH_ID":       "",
				"CLOUDNS_AUTH_PASSWORD": "",
			},
			expected: "ClouDNS: some credentials information are missing: CLOUDNS_AUTH_ID,CLOUDNS_AUTH_PASSWORD",
		},
		{
			desc: "missing auth-id",
			envVars: map[string]string{
				"CLOUDNS_AUTH_ID":       "",
				"CLOUDNS_AUTH_PASSWORD": "456",
			},
			expected: "ClouDNS: some credentials information are missing: CLOUDNS_AUTH_ID",
		},
		{
			desc: "missing auth-password",
			envVars: map[string]string{
				"CLOUDNS_AUTH_ID":       "123",
				"CLOUDNS_AUTH_PASSWORD": "",
			},
			expected: "ClouDNS: some credentials information are missing: CLOUDNS_AUTH_PASSWORD",
		},
		{
			desc: "invalid auth-id-type",
			envVars: map[string]string{
				"CLOUDNS_AUTH_ID":       "123",
				"CLOUDNS_AUTH_ID_TYPE":  "something",
				"CLOUDNS_AUTH_PASSWORD": "456",
			},
			expected: "ClouDNS auth id type is not valid. Expected one of 'auth-id' or 'sub-auth-id' but was: 'something'",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			applyEnv(t, test.envVars)

			p, err := NewDNSProvider()

			if len(test.expected) == 0 {
				require.NoError(t, err)
				require.NotNil(t, p)
				require.NotNil(t, p.config)
				require.NotNil(t, p.client)
			} else {
				require.EqualError(t, err, test.expected)
			}
		})
	}
}

func TestNewDNSProviderConfig(t *testing.T) {
	testCases := []struct {
		desc         string
		authID       string
		authPassword string
		expected     string
	}{
		{
			desc:         "success",
			authID:       "123",
			authPassword: "456",
		},
		{
			desc:     "missing credentials",
			expected: "ClouDNS: credentials missing: authID",
		},
		{
			desc:         "missing auth-id",
			authPassword: "456",
			expected:     "ClouDNS: credentials missing: authID",
		},
		{
			desc:     "missing auth-password",
			authID:   "123",
			expected: "ClouDNS: credentials missing: authPassword",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			config := NewDefaultConfig()
			config.AuthID = test.authID
			config.AuthPassword = test.authPassword

			p, err := NewDNSProviderConfig(config)

			if len(test.expected) == 0 {
				require.NoError(t, err)
				require.NotNil(t, p)
				require.NotNil(t, p.config)
				require.NotNil(t, p.client)
			} else {
				require.EqualError(t, err, test.expected)
			}
		})
	}
}

func TestLivePresent(t *testing.T) {
	if !isLiveTest() {
		t.Skip("skipping live test")
	}

	provider, err := NewDNSProvider()
	require.NoError(t, err)

	err = provider.Present(liveEnv[domainEnvVar], "123d==")
	require.NoError(t, err)
}

func TestLiveCleanUp(t *testing.T) {
	if !isLiveTest() {
		t.Skip("skipping live test")
	}

	provider, err := NewDNSProvider()
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	err = provider.CleanUp(liveEnv[domainEnvVar], "123d==")
	require.NoError(t, err)
}

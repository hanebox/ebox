package feature

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"ebox/app/facades"
	"ebox/app/services/proxy"
	"ebox/app/services/settings"
	_ "ebox/tests"
)

func TestComposeFileConfiguresTraefik(t *testing.T) {
	compose := proxy.TraefikService().ComposeFile(proxy.Settings{
		ACMEEmail: "ops@example.com",
		HTTPPort:  8080,
		HTTPSPort: 8443,
	})

	require.Contains(t, compose, "image: traefik:v3")
	require.Contains(t, compose, `container_name: ebox-traefik`)
	require.Contains(t, compose, `"8080:80"`)
	require.Contains(t, compose, `"8443:443"`)
	require.Contains(t, compose, `"127.0.0.1:8080:8080"`)
	require.Contains(t, compose, `--providers.docker.exposedbydefault=false`)
	require.Contains(t, compose, `--providers.file.directory=/etc/traefik/dynamic`)
	require.Contains(t, compose, `--certificatesresolvers.letsencrypt.acme.email=ops@example.com`)
	require.Contains(t, compose, `--certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web`)
	require.Contains(t, compose, `/var/run/docker.sock:/var/run/docker.sock:ro`)
	require.Contains(t, compose, `ebox.managed: "true"`)
	require.Contains(t, compose, `ebox.resource-type: "proxy"`)
	require.Contains(t, compose, `host.docker.internal:host-gateway`)
	require.Contains(t, compose, "external: true")
}

func TestComposeFileFallsBackToDefaultEmail(t *testing.T) {
	compose := proxy.TraefikService().ComposeFile(proxy.Settings{})

	require.Contains(t, compose, "--certificatesresolvers.letsencrypt.acme.email=admin@example.com")
	require.Contains(t, compose, `"80:80"`)
	require.Contains(t, compose, `"443:443"`)
}

func TestWriteConfigurationManagesPanelRoute(t *testing.T) {
	require.NoError(t, facades.Artisan().Call("migrate"))

	settingsService := settings.Default()
	original := settingsService.Get("panel_domain", "")
	t.Cleanup(func() { _ = settingsService.SetMany(map[string]string{"panel_domain": original}) })

	traefikDir := facades.Config().GetString("ebox.traefik_dir")
	panelPath := filepath.Join(traefikDir, "dynamic", "panel.yml")
	port := facades.Config().GetString("ebox.port", "8000")

	// With a panel domain the panel is published through the proxy.
	require.NoError(t, settingsService.SetMany(map[string]string{"panel_domain": "panel.test.example"}))
	require.NoError(t, proxy.TraefikService().WriteConfiguration())

	content, err := os.ReadFile(panelPath)
	require.NoError(t, err)
	require.Contains(t, string(content), "ebox-panel")
	require.Contains(t, string(content), "Host(`panel.test.example`)")
	require.Contains(t, string(content), "host.docker.internal:"+port)
	require.Contains(t, string(content), "certResolver: letsencrypt")

	compose, err := os.ReadFile(filepath.Join(traefikDir, "docker-compose.yml"))
	require.NoError(t, err)
	require.Contains(t, string(compose), "ebox-traefik")

	acme, err := os.ReadFile(filepath.Join(traefikDir, "acme", "acme.json"))
	require.NoError(t, err)
	require.Equal(t, "{}", string(acme))

	// Without a panel domain the route is withdrawn.
	require.NoError(t, settingsService.SetMany(map[string]string{"panel_domain": ""}))
	require.NoError(t, proxy.TraefikService().WriteConfiguration())

	_, statErr := os.Stat(panelPath)
	require.True(t, os.IsNotExist(statErr), "panel route should be removed without a panel domain")
}

func TestMiddlewareLifecycle(t *testing.T) {
	require.NoError(t, facades.Artisan().Call("migrate"))

	name := "test-auth-" + strings.ReplaceAll(time.Now().UTC().Format("150405.000"), ".", "")
	t.Cleanup(func() { _ = proxy.Default().DeleteMiddleware(t.Context(), name) })

	require.NoError(t, proxy.Default().SaveMiddleware(t.Context(), proxy.Middleware{
		Name: name,
		Type: "basicAuth",
		Config: map[string]any{
			"users": []string{"admin:$2y$10$abcdefghijklmnopqrstuv"},
		},
	}))

	list, err := proxy.Default().Middlewares(t.Context())
	require.NoError(t, err)

	found := false
	for _, middleware := range list {
		if middleware.Name != name {
			continue
		}

		found = true
		require.Equal(t, "basicAuth", middleware.Type)
		require.True(t, middleware.Editable)
		require.Equal(t, "file", middleware.Provider)
		require.Equal(t, []string{"admin:$2y$10$abcdefghijklmnopqrstuv"}, middleware.Config["users"])
	}
	require.True(t, found, "saved middleware should be listed")

	require.NoError(t, proxy.Default().DeleteMiddleware(t.Context(), name))

	after, err := proxy.Default().Middlewares(t.Context())
	require.NoError(t, err)
	for _, middleware := range after {
		require.NotEqual(t, name, middleware.Name)
	}
}

func TestCertificatesFromACMEStore(t *testing.T) {
	directory := filepath.Join(facades.Config().GetString("ebox.traefik_dir"), "acme")
	require.NoError(t, os.MkdirAll(directory, 0o755))

	certificatePEM := generateCertificate(t, "app.example.com")

	store := map[string]any{
		"letsencrypt": map[string]any{
			"Certificates": []map[string]any{
				{
					"domain":      map[string]any{"main": "app.example.com", "sans": []string{"app.example.com", "www.app.example.com"}},
					"certificate": []byte(certificatePEM),
					"key":         []byte("not-used"),
				},
			},
		},
	}

	content, err := json.Marshal(store)
	require.NoError(t, err)

	path := filepath.Join(directory, "acme.json")
	original, readErr := os.ReadFile(path)
	require.NoError(t, os.WriteFile(path, content, 0o600))
	t.Cleanup(func() {
		if readErr == nil {
			_ = os.WriteFile(path, original, 0o600)
		}
	})

	certificates, err := proxy.Default().Certificates(t.Context())
	require.NoError(t, err)
	require.Len(t, certificates, 1)

	certificate := certificates[0]
	require.Equal(t, "app.example.com", certificate.Domain)
	require.Equal(t, []string{"app.example.com", "www.app.example.com"}, certificate.SANs)
	require.Equal(t, "letsencrypt", certificate.ResolvedVia)
	require.True(t, certificate.NotAfter.After(time.Now()))
	require.Greater(t, certificate.DaysLeft, 0)
}

func generateCertificate(t *testing.T, host string) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: host},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(90 * 24 * time.Hour),
		DNSNames:     []string{host},
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	require.NoError(t, err)

	encoded := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	require.NotEmpty(t, encoded)

	return encoded
}

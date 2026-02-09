package tests

import (
	"ActQABot/conf"
	"ActQABot/pkg/hosts"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
)

func setupTestEnv(t *testing.T) {
	t.Helper()

	t.Setenv("HOST_CONF", "hosts.example.yaml")
	t.Setenv("GITHUB_TOKEN", "test-token")
	conf.NewEnviron(&conf.GeneralEnvironments)

	var err error
	conf.Hosts, err = conf.NewHostsEnvironment(conf.GeneralEnvironments.HostConf)
	if err != nil {
		t.Fatalf("failed to init conf.Hosts: %v", err)
	}
	hosts.HostAvbl = hosts.NewAvailability(conf.Hosts)
}

func generateRSAPrivateKeyPEM(t *testing.T, filePath string, bits int) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	pemBytes := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		},
	)

	err = os.WriteFile(filePath, pemBytes, 0600)
	if err != nil {
		t.Fatalf("failed to write private key file: %v", err)
	}
}

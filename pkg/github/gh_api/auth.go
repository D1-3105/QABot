package gh_api

import (
	"ActQABot/conf"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
	"net/http"
	"os"
	"time"
)

var Authorize = authForRepo
var GenerateJWTToken = generateJWT
var GHClientConstructor = func(client *http.Client) *github.Client {
	return github.NewClient(client)
}

var installIdMemoized map[string]int64

func loadPrivateKey(pemFile string) (*rsa.PrivateKey, error) {
	bytes, err := os.ReadFile(pemFile)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(bytes)
	if block == nil {
		return nil, fmt.Errorf("no valid PEM data found")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func generateJWT(privateKey *rsa.PrivateKey, appID string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    appID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

func makeClient(jwtToken string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{TokenType: "Bearer", AccessToken: jwtToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := GHClientConstructor(tc)
	return client
}

func getInstallIDFromRepo(client *github.Client, owner, repo string) (int64, error) {

	installation, _, err := client.Apps.FindRepositoryInstallation(context.Background(), owner, repo)
	if err != nil {
		return 0, err
	}
	return installation.GetID(), nil
}

func getInstallationToken(client *github.Client, installID int64) (*github.InstallationToken, error) {
	ctx := context.Background()
	token, _, err := client.Apps.CreateInstallationToken(ctx, installID, nil)
	return token, err
}

func authForRepo(ghEnv conf.GithubAPIEnvironment, owner, repo string) (*github.InstallationToken, error) {
	pk, err := loadPrivateKey(ghEnv.PrivateKeyPath)
	if err != nil {
		return nil, err
	}
	tok, err := GenerateJWTToken(pk, ghEnv.AppID)
	if err != nil {
		return nil, err
	}
	client := makeClient(tok)
	installID, ok := installIdMemoized[owner+"/"+repo]
	if !ok {
		installID, err = getInstallIDFromRepo(client, owner, repo)
		if err != nil {
			return nil, err
		}
	}
	return getInstallationToken(client, installID)
}

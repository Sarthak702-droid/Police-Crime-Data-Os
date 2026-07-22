package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWKSCacheReturnsRSAPublicKey(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	n := base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1})
	cache := newJWKSCache("https://keycloak.test/certs")
	cache.http.Transport = middlewareRoundTrip(func(request *http.Request) (*http.Response, error) {
		body := fmt.Sprintf(`{"keys":[{"kid":"key-1","kty":"RSA","alg":"RS256","n":"%s","e":"%s"}]}`, n, e)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
	})
	token := jwt.New(jwt.SigningMethodRS256)
	token.Header["kid"] = "key-1"
	key, err := cache.key(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, ok := key.(*rsa.PublicKey)
	if !ok || publicKey.N.Cmp(privateKey.PublicKey.N) != 0 {
		t.Fatal("unexpected RSA public key")
	}
}

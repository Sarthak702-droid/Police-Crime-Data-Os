package middleware

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type jwksCache struct {
	url       string
	http      *http.Client
	mu        sync.RWMutex
	keys      map[string]interface{}
	expiresAt time.Time
}

func newJWKSCache(url string) *jwksCache {
	return &jwksCache{url: url, http: &http.Client{Timeout: 10 * time.Second}, keys: map[string]interface{}{}}
}

func (c *jwksCache) key(ctx context.Context, token *jwt.Token) (interface{}, error) {
	if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
		return nil, errors.New("OIDC token must use RS256")
	}
	kid, _ := token.Header["kid"].(string)
	if kid == "" {
		return nil, errors.New("OIDC token is missing kid")
	}
	c.mu.RLock()
	key, ok := c.keys[kid]
	fresh := time.Now().Before(c.expiresAt)
	c.mu.RUnlock()
	if ok && fresh {
		return key, nil
	}
	if err := c.refresh(ctx, !ok); err != nil {
		return nil, err
	}
	c.mu.RLock()
	key, ok = c.keys[kid]
	c.mu.RUnlock()
	if !ok {
		return nil, errors.New("OIDC signing key not found")
	}
	return key, nil
}

func (c *jwksCache) refresh(ctx context.Context, force bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !force && time.Now().Before(c.expiresAt) && len(c.keys) > 0 {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("JWKS request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("JWKS endpoint returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	var set struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Alg string `json:"alg"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(body, &set); err != nil {
		return err
	}
	keys := map[string]interface{}{}
	for _, item := range set.Keys {
		if item.Kty != "RSA" || item.Kid == "" || item.N == "" || item.E == "" {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(item.N)
		if err != nil {
			continue
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(item.E)
		if err != nil {
			continue
		}
		exponent := 0
		for _, b := range eBytes {
			exponent = exponent<<8 + int(b)
		}
		if exponent <= 0 {
			continue
		}
		keys[item.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: exponent}
	}
	if len(keys) == 0 {
		return errors.New("JWKS endpoint returned no usable RSA keys")
	}
	c.keys = keys
	c.expiresAt = time.Now().Add(15 * time.Minute)
	return nil
}

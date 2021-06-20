// Package auth provides authentication and authorization support.
package auth

import (
	"crypto/rsa"

	"github.com/pkg/errors"

	"github.com/dgrijalva/jwt-go/v4"
)

// These are the expected values for Claims.Roles.
const (
	RoleAdmin = "ADMIN"
	RoleUser  = "USER"
)

// ctxKey represents the type of value for the context key.
type ctxKey int

// Key is used to store/retrieve a Claims value from a context.Context.
const Key ctxKey = 1

// Claims represents the authorization claims transmitted via a JWT.
type Claims struct {
	jwt.StandardClaims
	Roles []string `json:"roles"`
}

// Authorize returns true if the claims has at least one of the provided roles.
func (c Claims) Authorize(roles ...string) bool {
	for _, has := range c.Roles {
		for _, want := range roles {
			if has == want {
				return true
			}
		}
	}
	return false
}

// Keys represents an in memory store of keys.
type Keys map[string]*rsa.PrivateKey

type PublickeyLookup func (kid string) (*rsa.PublicKey, error)

// Auth is used to authenticate clients. It can generate a token for a
// set of user claims and recreate the claims by parsing the token.
type Auth struct {
	algorithm string
	keyFunc   func(t *jwt.Token) (interface{}, error)
	parser    *jwt.Parser
	keys Keys
}

// New creates an Auth to support authentication/authorization.
func New(algorithm string, lookup PublickeyLookup, keys Keys) (*Auth, error) {

	if jwt.GetSigningMethod(algorithm) == nil {
		return nil, errors.Errorf("unknown algorithm %v", algorithm)
	}

	keyFunc := func(t *jwt.Token) (interface{}, error) {
		kid, ok := t.Header["kid"]
		if !ok {
			return nil, errors.New("missing key id (kid) in token header")
		}
		kidID, ok := kid.(string)
		if !ok {
			return nil, errors.New("user token key id (kid) must be string")
		}
		return lookup(kidID)
	}

	// Create the token parser to use. The algorithm used to sign the JWT must be
	// validated to avoid a critical vulnerability:
	// https://auth0.com/blog/critical-vulnerabilities-in-json-web-token-libraries/
	parser := jwt.NewParser(jwt.WithValidMethods([]string{algorithm}), jwt.WithAudience("student"))

	a := Auth{
		algorithm: algorithm,
		keyFunc:   keyFunc,
		parser:    parser,
		keys: keys,
	}

	return &a, nil
}

// AddKey adds a private key and combination kid id to our local store.
func (a *Auth) AddKey(privateKey *rsa.PrivateKey, kid string) {
	a.keys[kid] = privateKey
}

// RemoveKey removes a private key and combination kid id to our local store
func (a *Auth) RemoveKey(kid string) {
	delete(a.keys, kid)
}

// GenerateToken generates a signed JWT token string representing the user Claims.
func (a *Auth) GenerateToken(kid string, claims Claims) (string, error) {

	method := jwt.GetSigningMethod(a.algorithm)
	token := jwt.NewWithClaims(method, claims)
	token.Header["kid"] = kid

	privateKey, ok := a.keys[kid]
	if !ok {
		return "", errors.New("kid lookup failed")
	}

	str, err := token.SignedString(privateKey)
	if err != nil {
		return "", errors.Wrap(err, "signing token")
	}

	return str, nil
}

// ValidateToken recreates the Claims that were used to generate a token. It
// verifies that the token was signed using our key.
func (a *Auth) ValidateToken(tokenStr string) (Claims, error) {

	var claims Claims
	token, err := a.parser.ParseWithClaims(tokenStr, &claims, a.keyFunc)
	if err != nil {
		return Claims{}, errors.Wrap(err, "parsing token")
	}

	if !token.Valid {
		return Claims{}, errors.New("invalid token")
	}

	return claims, nil

}

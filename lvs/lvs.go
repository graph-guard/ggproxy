package lvs

import (
	"crypto"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"errors"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

type Type uint16
type Plan uint16

type JwtClaim struct {
	jwt.StandardClaims
	Type Type `json:"type"`
	Plan Plan `json:"plan"`
}

const (
	Beta Type = iota
	Individual
	Commercial
)

const (
	Tiny Plan = iota
	Small
	Medium
	Big
	Unlimited
)

// Encoded public key, uniq per client
var PublicKey string

var ErrFailParseClaims = errors.New("fail to parse claims")
var ErrLicenseExpire = errors.New("license expire")

// ValidateLicenseToken verifies the license and return license key parameters as claims
func ValidateLicenseToken(licenseToken string) (*JwtClaim, error) {
	decodedPublicKey, err := decodePublicKey([]byte(PublicKey))
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(
		licenseToken,
		&JwtClaim{},
		func(token *jwt.Token) (interface{}, error) {
			return decodedPublicKey, nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JwtClaim)
	if !ok {
		return nil, ErrFailParseClaims
	}

	if claims.ExpiresAt < time.Now().Local().Unix() {
		return nil, ErrLicenseExpire
	}

	return claims, nil
}

func decodePublicKey(pemEncoded []byte) (crypto.PublicKey, error) {
	blockPub, _ := pem.Decode(pemEncoded)
	x509Encoded := blockPub.Bytes
	genericPublicKey, err := x509.ParsePKIXPublicKey(x509Encoded)
	if err != nil {
		return nil, err
	}
	publicKey := genericPublicKey.(crypto.PublicKey)

	return publicKey, nil
}

package lvs

import (
	"crypto"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"errors"

	jwt "github.com/dgrijalva/jwt-go"
)

type Type uint16
type Plan uint16

type LicenseTokenClaim struct {
	jwt.StandardClaims
	Type Type `json:"type"`
	Plan Plan `json:"plan"`
}

const (
	Beta Type = iota
	Community
	Commercial
)

const (
	Tiny Plan = iota
	Small
	Medium
	Big
	Unlimited
)

// Encoded public key
var PublicKey string

var ErrFailParseClaims = errors.New("failed to parse license token claims")
var ErrLicenseExpired = errors.New("license expired")
var ErrLicenseMalformed = errors.New("license token malformed or empty")
var ErrNoPEMBlock = errors.New("no valid PEM block in public key")

// ValidateLicenseToken verifies the license and return license key parameters as claims.
func ValidateLicenseToken(licenseToken string) (*LicenseTokenClaim, error) {
	if PublicKey == "" {
		panic("missing public key")
	}

	decodedPublicKey, err := decodePublicKey([]byte(PublicKey))
	if err != nil {
		panic(err)
	}

	token, err := jwt.ParseWithClaims(
		licenseToken,
		&LicenseTokenClaim{},
		func(token *jwt.Token) (interface{}, error) {
			return decodedPublicKey, nil
		},
	)

	e, ok := err.(*jwt.ValidationError)
	if ok {
		if e.Errors&jwt.ValidationErrorExpired != 0 {
			return nil, ErrLicenseExpired
		}
		if e.Errors&jwt.ValidationErrorMalformed != 0 {
			return nil, ErrLicenseMalformed
		}
	}

	claims, ok := token.Claims.(*LicenseTokenClaim)
	if !ok {
		return nil, ErrFailParseClaims
	}

	return claims, err
}

func decodePublicKey(pemEncoded []byte) (crypto.PublicKey, error) {
	block, _ := pem.Decode(pemEncoded)
	if block == nil {
		return nil, ErrNoPEMBlock
	}
	x509Encoded := block.Bytes
	genericPublicKey, err := x509.ParsePKIXPublicKey(x509Encoded)
	if err != nil {
		return nil, err
	}
	publicKey := genericPublicKey.(crypto.PublicKey)

	return publicKey, nil
}

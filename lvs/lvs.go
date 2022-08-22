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

type LicenseTokenClaim struct {
	jwt.StandardClaims
	Type Type   `json:"type"`
	Plan Plan   `json:"plan"`
	Pub  []byte `json:"pub"`
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

var ErrFailParseClaims = errors.New("fail to parse claims")
var ErrLicenseExpired = errors.New("license expired")

// ValidateLicenseToken verifies the license and return license key parameters as claims.
func ValidateLicenseToken(timeNow time.Time, licenseToken string) (*LicenseTokenClaim, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(licenseToken, &LicenseTokenClaim{})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*LicenseTokenClaim)
	if !ok {
		return nil, ErrFailParseClaims
	}
	decodedPublicKey, err := decodePublicKey(claims.Pub)
	if err != nil {
		return nil, err
	}

	token, err = jwt.Parse(
		licenseToken,
		func(token *jwt.Token) (interface{}, error) {
			return decodedPublicKey, nil
		},
	)

	e, ok := err.(*jwt.ValidationError)
	if ok {
		if e.Errors&jwt.ValidationErrorExpired != 0 {
			return nil, ErrLicenseExpired
		}
	}

	return claims, err
}

func decodePublicKey(pemEncoded []byte) (crypto.PublicKey, error) {
	block, _ := pem.Decode(pemEncoded)
	x509Encoded := block.Bytes
	genericPublicKey, err := x509.ParsePKIXPublicKey(x509Encoded)
	if err != nil {
		return nil, err
	}
	publicKey := genericPublicKey.(crypto.PublicKey)

	return publicKey, nil
}

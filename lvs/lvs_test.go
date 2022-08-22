package lvs_test

import (
	"crypto"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/graph-guard/ggproxy/lvs"
	"github.com/stretchr/testify/require"
)

var privateKey = `
-----BEGIN EC PRIVATE KEY-----
MIHcAgEBBEIBK43Z3ATV+af8U8iMBODHssl4FsSvL70DePBubkthHlltuVPxu29X
T7/Q5zSICfpRD0Q8F9lGxum5KPl4T/n6IM2gBwYFK4EEACOhgYkDgYYABAHtTblG
M/FaKHDkVrBOSJ2SJe7+Spyxbn7DQOfZ0B4dVVALGc5j/G+TqeYt6DVO1GEOHL3/
lMy0L827kdCU5iZopgHJrjeaM38n4HmG/dEh4x7R3P+rNDV0NQ7EJ4W8dY+HwtDb
ripL46GdA8fVwDgom/qe4btdpGJBQcJrmURLxQjAXw==
-----END EC PRIVATE KEY-----
`

var publicKey = `
-----BEGIN PUBLIC KEY-----
MIGbMBAGByqGSM49AgEGBSuBBAAjA4GGAAQB7U25RjPxWihw5FawTkidkiXu/kqc
sW5+w0Dn2dAeHVVQCxnOY/xvk6nmLeg1TtRhDhy9/5TMtC/Nu5HQlOYmaKYBya43
mjN/J+B5hv3RIeMe0dz/qzQ1dDUOxCeFvHWPh8LQ264qS+OhnQPH1cA4KJv6nuG7
XaRiQUHCa5lES8UIwF8=
-----END PUBLIC KEY-----
`

func TestVerifyLicenceKey(t *testing.T) {
	decodedLicenseKey, err := GenerateLicenseToken(1)
	require.NoError(t, err)
	require.NotEmpty(t, decodedLicenseKey)

	claims, err := lvs.ValidateLicenseToken(decodedLicenseKey)
	require.NoError(t, err)
	require.NotNil(t, claims)
}

func GenerateLicenseToken(expirationHours int64) (signedToken string, err error) {
	claims := &lvs.LicenseTokenClaim{
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  time.Now().Local().Unix(),
			ExpiresAt: time.Now().Local().Add(time.Hour * time.Duration(expirationHours)).Unix(),
		},
		Type: lvs.Beta,
		Plan: lvs.Unlimited,
		Pub:  []byte(publicKey),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES512, claims)

	decodedPrivateKey, err := decodePrivateKey([]byte(privateKey))
	if err != nil {
		return "", err
	}

	signedToken, err = token.SignedString(decodedPrivateKey)
	if err != nil {
		return
	}

	return
}

func decodePrivateKey(pemEncoded []byte) (crypto.PrivateKey, error) {
	block, _ := pem.Decode(pemEncoded)
	x509Encoded := block.Bytes
	privateKey, err := x509.ParseECPrivateKey(x509Encoded)

	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

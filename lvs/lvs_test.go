package lvs_test

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"testing"

	"github.com/graph-guard/gguard-proxy/lvs"
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

var fingerprint = "62f8216d-7b07-4a39-a0cb-b33635de2e55"

func TestVerifyLicenceKey(t *testing.T) {
	decodedLicenseKey, err := GenerateLicenseKey()
	require.NoError(t, err)
	require.NotEmpty(t, decodedLicenseKey)

	lvs.Fingerprint = fingerprint
	lvs.PublicKey = publicKey

	valid, err := lvs.VerifyLicenceKey(decodedLicenseKey)
	require.NoError(t, err)
	require.Equal(t, true, valid)
}

func GenerateLicenseKey() ([]byte, error) {
	hash, err := lvs.CalculateExecutableHash()
	if err != nil {
		return nil, err
	}

	decodedPrivateKey, err := decodePrivateKey([]byte(privateKey))
	if err != nil {
		return nil, err
	}

	key, err := ecdsa.SignASN1(rand.Reader, decodedPrivateKey, hash)
	if err != nil {
		return nil, err
	}

	return key, nil
}

func decodePrivateKey(pemEncoded []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(pemEncoded)
	x509Encoded := block.Bytes
	privateKey, err := x509.ParseECPrivateKey(x509Encoded)

	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

package lvs_test

import (
	_ "embed"
	"testing"
	"time"

	"github.com/google/uuid"
	blvs "github.com/graph-guard/backend/lvs"
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

func TestVerifyLicenseToken(t *testing.T) {
	decodedLicenseToken, err := blvs.GenerateLicenseToken(
		time.Now().Local(),
		time.Now().Local().Add(time.Hour),
		lvs.Beta,
		lvs.Unlimited,
		uuid.New(),
		[]byte(privateKey),
	)
	require.NoError(t, err)
	require.NotEmpty(t, decodedLicenseToken)

	lvs.PublicKey = publicKey

	claims, err := lvs.ValidateLicenseToken(
		string(decodedLicenseToken),
	)

	require.NoError(t, err)
	require.NotNil(t, claims)
}

func TestLicenseTokenExpired(t *testing.T) {
	decodedLicenseToken, err := blvs.GenerateLicenseToken(
		time.Now().Local(),
		time.Now().Local().Add(-time.Hour),
		lvs.Beta,
		lvs.Unlimited,
		uuid.New(),
		[]byte(privateKey),
	)
	require.NoError(t, err)
	require.NotEmpty(t, decodedLicenseToken)

	lvs.PublicKey = publicKey

	claims, err := lvs.ValidateLicenseToken(
		string(decodedLicenseToken),
	)

	require.Error(t, lvs.ErrLicenseExpired, err)
	require.Nil(t, claims)
}

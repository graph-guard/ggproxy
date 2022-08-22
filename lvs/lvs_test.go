package lvs_test

import (
	_ "embed"
	"fmt"
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

func TestVerifyLicenceKey(t *testing.T) {
	decodedLicenseKey, err := blvs.GenerateLicenseToken(
		Time(t, "2022-01-01T14:00:00Z"),
		1,
		lvs.Beta,
		lvs.Unlimited,
		uuid.New(),
		[]byte(privateKey),
		[]byte(publicKey),
	)
	require.NoError(t, err)
	require.NotEmpty(t, decodedLicenseKey)

	claims, err := lvs.ValidateLicenseToken(
		Time(t, "2022-01-01T14:00:00Z"),
		string(decodedLicenseKey),
	)

	fmt.Printf("%#v", err)

	require.NoError(t, err)
	require.NotNil(t, claims)
}

func Time(t *testing.T, s string) time.Time {
	tm, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err)
	return tm
}

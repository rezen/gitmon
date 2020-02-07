package gitmon

import (
	qrcode "github.com/skip2/go-qrcode"

	"crypto/rand"
	"encoding/base32"
	// "encoding/base64"

	// "fmt"
	"github.com/dgryski/dgoogauth"
	"strings"
)

/*
type VerifyUserOTP struct {
	User User
}
*/

func VerifyOTPToken(token string, secret string) (bool, error) {
	otp := &dgoogauth.OTPConfig{
		Secret:      strings.TrimSpace(secret),
		WindowSize:  3,
		HotpCounter: 0,
	}
	return otp.Authenticate(strings.TrimSpace(token))
}


func GenerateOTPSecret() string {
	dictionary := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var secret = make([]byte, 10)
	rand.Read(secret)
	for k, v := range secret {
		secret[k] = dictionary[v%byte(len(dictionary))]
	}
	return base32.StdEncoding.EncodeToString(secret)
}

type ProvisionOTP struct {
	Link string
	Code []byte
	Dataurl string
}

func GenerateOTPProvision(username string) (*ProvisionOTP, error) {
	secret := GenerateOTPSecret()
	// https://github.com/google/google-authenticator/wiki/Key-Uri-Format
	otp := &dgoogauth.OTPConfig{
		Secret:      strings.TrimSpace(secret),
		WindowSize:  3,
		HotpCounter: 0,
	}

	provision := &ProvisionOTP{}
	provision.Link = otp.ProvisionURIWithIssuer(username, "gitmon")
	code, err := qrcode.Encode(provision.Link, qrcode.Medium, 256)
	provision.Code = code
	// provision.Dataurl = "data:image/png;base64," + base64.StdEncoding.EncodeToString(provision.Code)
	return provision, err
}
package utils

import "regexp"

//goland:noinspection SpellCheckingInspection
var regexpValoperAddress = regexp.MustCompile(`^[a-z\d]+valoper1[qpzry9x8gf2tvdw0s3jn54khce6mua7l]{38,}$`)

//goland:noinspection SpellCheckingInspection
func IsValoperAddressFormat(address string) bool {
	return regexpValoperAddress.MatchString(address)
}

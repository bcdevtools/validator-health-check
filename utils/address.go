package utils

import "strings"

// GetValconsHrpFromValoperHrp predicts valcons hrp from valoper hrp
func GetValconsHrpFromValoperHrp(valoper string) (valconsHrp string, success bool) {
	lastIndex1 := strings.LastIndex(valoper, "1")
	if lastIndex1 < 0 {
		return "", false
	}
	if lastIndex1 == 0 {
		return "", false
	}

	valoperHrp := valoper[:lastIndex1]
	if !strings.HasSuffix(valoperHrp, "valoper") {
		return "", false
	}

	// replace suffix by 'valcons'
	valconsHrp = valoperHrp[:len(valoperHrp)-len("valoper")] + "valcons"
	success = true
	return
}

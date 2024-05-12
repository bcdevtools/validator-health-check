package validator_address_registry

import (
	"sync"
)

var mutex sync.RWMutex

//goland:noinspection SpellCheckingInspection
var (
	globalValoperToValcons map[string]map[string]string // chainName -> [valoper -> valcons]
	globalValconsToValoper map[string]map[string]string // chainName -> [valcons -> valoper]
)

func RegisterPairValAddressWL(chainName string, valoper string, valcons string) {
	mutex.Lock()
	defer mutex.Unlock()

	if _, found := globalValoperToValcons[chainName]; !found {
		globalValoperToValcons[chainName] = make(map[string]string)
	}
	globalValoperToValcons[chainName][valoper] = valcons

	if _, found := globalValconsToValoper[chainName]; !found {
		globalValconsToValoper[chainName] = make(map[string]string)
	}
	globalValconsToValoper[chainName][valcons] = valoper
}

// TODO remove if not use
//
//goland:noinspection SpellCheckingInspection
func GetValconsByValoperRL(chainName string, valoper string) (valcons string, found bool) {
	mutex.RLock()
	defer mutex.RUnlock()

	var mapper map[string]string
	mapper, found = globalValoperToValcons[chainName]
	if !found {
		return
	}

	valcons, found = mapper[valoper]
	return
}

// TODO remove if not use
//
//goland:noinspection SpellCheckingInspection
func GetValoperByValconsRL(chainName string, valcons string) (valoper string, found bool) {
	mutex.RLock()
	defer mutex.RUnlock()

	var mapper map[string]string
	mapper, found = globalValconsToValoper[chainName]
	if !found {
		return
	}

	valoper, found = mapper[valcons]
	return
}

func init() {
	globalValoperToValcons = make(map[string]map[string]string)
	globalValconsToValoper = make(map[string]map[string]string)
}

package validator_address_registry

import (
	"sync"
)

var mutex sync.RWMutex

//goland:noinspection SpellCheckingInspection
var (
	globalValoperToValcons map[string]map[string]string // chainName -> [valoper -> valcons]
	globalValconsToValoper map[string]map[string]string // chainName -> [valcons -> valoper]
	globalValoperToAddress map[string]string            // valoper -> address
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

func RegisterPairValAddressToAddressWL(valoper string, addr string) {
	mutex.Lock()
	defer mutex.Unlock()

	globalValoperToAddress[valoper] = addr
}

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

func GetAddressByValoperRL(valoper string) (addr string, found bool) {
	mutex.RLock()
	defer mutex.RUnlock()

	addr, found = globalValoperToAddress[valoper]
	return
}

func init() {
	globalValoperToValcons = make(map[string]map[string]string)
	globalValconsToValoper = make(map[string]map[string]string)
	globalValoperToAddress = make(map[string]string)
}

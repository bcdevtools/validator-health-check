package rpc_client_registry

import (
	"github.com/EscanBE/go-lib/logging"
	"sync"
)

var mutex sync.RWMutex
var globalRpcToClient map[string]RpcClient

func GetRpcClientByEndpointWL(endpoint string, logger logging.Logger) (RpcClient, error) {
	if endpoint == "" {
		panic("empty rpc endpoint")
	}

	client, found, err := getRpcClientByEndpointRL(endpoint, true)
	if err != nil {
		return nil, err
	}
	if found {
		return client, nil
	}

	mutex.Lock()
	defer mutex.Unlock()

	// double check
	client, found, err = getRpcClientByEndpointRL(endpoint, false)
	if err != nil {
		return nil, err
	}
	if found {
		return client, nil
	}

	client, err = newRpcClient(endpoint)
	if err != nil {
		return nil, err
	}

	client = client.WithLogger(logger)

	globalRpcToClient[endpoint] = client
	return client, nil
}

func getRpcClientByEndpointRL(endpoint string, acquireRLock bool) (client RpcClient, found bool, err error) {
	if acquireRLock {
		mutex.RLock()
		defer mutex.RUnlock()
	}

	client, found = globalRpcToClient[endpoint]
	return
}

func init() {
	globalRpcToClient = make(map[string]RpcClient)
}

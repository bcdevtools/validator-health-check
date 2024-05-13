package rpc_client_registry

//goland:noinspection SpellCheckingInspection
import (
	"fmt"
	"github.com/EscanBE/go-lib/logging"
	"github.com/bcdevtools/validator-health-check/utils"
	"github.com/pkg/errors"
	httpclient "github.com/tendermint/tendermint/rpc/client/http"
	jsonrpcclient "github.com/tendermint/tendermint/rpc/jsonrpc/client"
	"net/http"
	"strings"
)

type RpcClient interface {
	GetWebsocketClient() *httpclient.HTTP
	WithLogger(logging.Logger) RpcClient
}

var _ RpcClient = &rpcClient{}

type rpcClient struct {
	logger logging.Logger

	httpClient      *http.Client
	websocketClient *httpclient.HTTP
}

func newRpcClient(endpoint string) (RpcClient, error) {
	httpEndpoint := utils.ReplaceAnySchemeWithHttp(endpoint)
	httpEndpoint = strings.TrimSuffix(httpEndpoint, "/")

	httpClient26657, websocketClient26657, err := getTendermintClient(httpEndpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Tendermint client")
	}

	return &rpcClient{
		httpClient:      httpClient26657,
		websocketClient: websocketClient26657,
	}, nil
}

func (r *rpcClient) GetWebsocketClient() *httpclient.HTTP {
	return r.websocketClient
}

func (r *rpcClient) WithLogger(logger logging.Logger) RpcClient {
	r.logger = logger
	return r
}

func getTendermintClient(rpc string) (*http.Client, *httpclient.HTTP, error) {
	httpClient26657, err := jsonrpcclient.DefaultHTTPClient(rpc)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to create HTTP client for %s", rpc)
	}
	httpTransport, ok := (httpClient26657.Transport).(*http.Transport)
	if !ok {
		return nil, nil, fmt.Errorf("failed to cast HTTP client transport to *http.Transport for %s", rpc)
	}
	httpTransport.MaxConnsPerHost = 20
	tendermintRpcHttpClient, err := httpclient.NewWithClient(rpc, "/websocket", httpClient26657)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to create Tendermint RPC client for %s", rpc)
	}
	err = tendermintRpcHttpClient.Start()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to start Tendermint RPC client for %s", rpc)
	}

	return httpClient26657, tendermintRpcHttpClient, nil
}

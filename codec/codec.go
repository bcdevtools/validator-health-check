package codec

import (
	sdkcodec "github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
)

var interfaceRegistry = codectypes.NewInterfaceRegistry()
var CryptoCodec = sdkcodec.NewProtoCodec(interfaceRegistry)

func init() {
	cryptocodec.RegisterInterfaces(interfaceRegistry)
}

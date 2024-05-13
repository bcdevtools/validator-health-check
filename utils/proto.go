package utils

//goland:noinspection SpellCheckingInspection
import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cmcryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	tmcrypto "github.com/tendermint/tendermint/crypto"
)

func FromAnyPubKeyToConsensusAddress(protoAny *codectypes.Any, codec codec.Codec) (consAddr sdk.ConsAddress, success bool) {
	var err error

	var cosmosPubKey cmcryptotypes.PubKey
	err = codec.UnpackAny(protoAny, &cosmosPubKey)
	if err == nil {
		return sdk.ConsAddress(cosmosPubKey.Address()), true
	}

	var tmPubKey tmcrypto.PubKey
	err = codec.UnpackAny(protoAny, &tmPubKey)
	if err == nil {
		return sdk.ConsAddress(tmPubKey.Address()), true
	}

	return nil, false
}

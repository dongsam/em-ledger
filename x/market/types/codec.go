// This software is Copyright (c) 2019-2020 e-Money A/S. It is not offered under an open source license.
//
// Please contact partners@e-money.com for licensing related questions.

package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/supply"
)

// Register concrete types on codec codec
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterConcrete(MsgAddOrder{}, "e-money/MsgAddOrder", nil)
	cdc.RegisterConcrete(MsgCancelReplaceOrder{}, "e-money/MsgCancelReplaceOrder", nil)
	cdc.RegisterConcrete(MsgCancelOrder{}, "e-money/MsgCancelOrder", nil)
}

// module codec
var ModuleCdc *codec.Codec

func init() {
	ModuleCdc = codec.New()
	RegisterCodec(ModuleCdc)
	codec.RegisterCrypto(ModuleCdc)
	auth.RegisterCodec(ModuleCdc)
	supply.RegisterCodec(ModuleCdc)

	ModuleCdc.Seal()
}

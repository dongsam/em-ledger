package authority

import (
	"fmt"

	"emoney/x/authority/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func newHandler(keeper Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {

		case types.MsgCreateIssuer:
			return handleMsgCreateIssuer(ctx, keeper, msg)
		case types.MsgDestroyIssuer:
			return handleMsgDestroyIssuer(ctx, keeper, msg)
		default:
			errMsg := fmt.Sprintf("Unrecognized inflation Msg type: %v", msg.Type())
			return sdk.ErrUnknownRequest(errMsg).Result()
		}
	}
}

func handleMsgDestroyIssuer(ctx sdk.Context, k Keeper, msg types.MsgDestroyIssuer) sdk.Result {
	err := k.DestroyIssuer(ctx, msg.Authority, msg.Issuer)
	if err != nil {
		return err.Result()
	}

	return sdk.Result{}
}

func handleMsgCreateIssuer(ctx sdk.Context, k Keeper, msg types.MsgCreateIssuer) sdk.Result {
	err := k.CreateIssuer(ctx, msg.Authority, msg.Issuer, msg.Denominations)
	if err != nil {
		return err.Result()
	}

	return sdk.Result{}
}

// This software is Copyright (c) 2019 e-Money A/S. It is not offered under an open source license.
//
// Please contact partners@e-money.com for licensing related questions.

package keeper

import (
	"fmt"
	"sort"

	"github.com/e-money/em-ledger/x/issuer/types"
	lp "github.com/e-money/em-ledger/x/liquidityprovider"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"
)

const (
	keyIssuerList = "issuers"
)

type Keeper struct {
	storeKey sdk.StoreKey
	lpKeeper lp.Keeper
	ik       types.InflationKeeper
}

func NewKeeper(storeKey sdk.StoreKey, lpk lp.Keeper, ik types.InflationKeeper) Keeper {
	return Keeper{
		storeKey: storeKey,
		lpKeeper: lpk,
		ik:       ik,
	}
}

func (k Keeper) IncreaseMintableAmountOfLiquidityProvider(ctx sdk.Context, liquidityProvider sdk.AccAddress, issuer sdk.AccAddress, mintableIncrease sdk.Coins) sdk.Result {
	logger := k.logger(ctx)

	i, err := k.mustBeIssuer(ctx, issuer)
	if err != nil {
		return types.ErrNotAnIssuer(issuer).Result()
	}

	for _, coin := range mintableIncrease {
		if !anyContained(i.Denoms, coin.Denom) {
			return types.ErrDoesNotControlDenomination(coin.Denom).Result()
		}
	}

	lpAcc := k.lpKeeper.GetLiquidityProviderAccount(ctx, liquidityProvider)
	if lpAcc == nil {
		logger.Info("Creating liquidity provider", "account", liquidityProvider, "increase", mintableIncrease)
		// Account was not previously a liquidity provider. Create it
		if res := k.lpKeeper.CreateLiquidityProvider(ctx, liquidityProvider, mintableIncrease); !res.IsOK() {
			return res
		}
	} else {
		logger.Info("Increasing liquidity provider mintable amount", "account", liquidityProvider, "increase", mintableIncrease)
		lpAcc.IncreaseMintableAmount(mintableIncrease)
		k.lpKeeper.SetLiquidityProviderAccount(ctx, lpAcc)
	}

	return sdk.Result{Events: ctx.EventManager().Events()}
}

func (k Keeper) DecreaseMintableAmountOfLiquidityProvider(ctx sdk.Context, liquidityProvider sdk.AccAddress, issuer sdk.AccAddress, mintableDecrease sdk.Coins) sdk.Result {
	logger := k.logger(ctx)

	i, err := k.mustBeIssuer(ctx, issuer)
	if err != nil {
		return types.ErrNotAnIssuer(issuer).Result()
	}

	for _, coin := range mintableDecrease {
		if !anyContained(i.Denoms, coin.Denom) {
			return types.ErrDoesNotControlDenomination(coin.Denom).Result()
		}
	}

	lpAcc := k.lpKeeper.GetLiquidityProviderAccount(ctx, liquidityProvider)
	if lpAcc == nil {
		return types.ErrNotLiquidityProvider(liquidityProvider).Result()
	}

	_, anyNegative := lpAcc.Mintable.SafeSub(mintableDecrease)
	if anyNegative {
		return types.ErrNegativeMintableBalance(lpAcc.GetAddress()).Result()
	}

	logger.Info("Liquidity provider mintable amount decreased", "account", liquidityProvider, "decrease", mintableDecrease)
	lpAcc.DecreaseMintableAmount(mintableDecrease)
	k.lpKeeper.SetLiquidityProviderAccount(ctx, lpAcc)

	return sdk.Result{Events: ctx.EventManager().Events()}

}

func (k Keeper) RevokeLiquidityProvider(ctx sdk.Context, liquidityProvider sdk.AccAddress, issuerAddress sdk.AccAddress) sdk.Result {
	issuer, err := k.mustBeIssuer(ctx, issuerAddress)
	if err != nil {
		return types.ErrNotAnIssuer(issuerAddress).Result()
	}

	lpAcc := k.lpKeeper.GetLiquidityProviderAccount(ctx, liquidityProvider)
	if lpAcc == nil {
		return types.ErrNotLiquidityProvider(liquidityProvider).Result()
	}

	newMintableAmount := lpAcc.Mintable
	for _, denom := range issuer.Denoms {
		newMintableAmount = removeDenom(newMintableAmount, denom)
	}

	if len(newMintableAmount) == len(lpAcc.Mintable) {
		// Nothing was changed. Issuer was not controlling this lp.
		return types.ErrNotLiquidityProvider(liquidityProvider).Result()
	}

	if len(newMintableAmount) == 0 {
		// Mintable amount is zero, so demote to ordinary account
		k.lpKeeper.RevokeLiquidityProviderAccount(ctx, lpAcc)
	} else {
		// This liquidity provider has been granted a mintable amounts from multiple issuers so some amount remain.
		lpAcc.Mintable = newMintableAmount
		k.lpKeeper.SetLiquidityProviderAccount(ctx, lpAcc)
	}

	return sdk.Result{Events: ctx.EventManager().Events()}
}

func (k Keeper) SetInflationRate(ctx sdk.Context, issuer sdk.AccAddress, inflationRate sdk.Dec, denom string) sdk.Result {
	_, err := k.mustBeIssuerOfDenom(ctx, issuer, denom)
	if err != nil {
		return types.ErrNotAnIssuer(issuer).Result()
	}

	return k.ik.SetInflation(ctx, inflationRate, denom)
}

func (k Keeper) logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) GetIssuers(ctx sdk.Context) (issuers []types.Issuer) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(keyIssuerList))
	if bz == nil {
		return
	}

	types.ModuleCdc.MustUnmarshalBinaryLengthPrefixed(bz, &issuers)
	return
}

func (k Keeper) setIssuers(ctx sdk.Context, issuers []types.Issuer) {
	store := ctx.KVStore(k.storeKey)
	bz := types.ModuleCdc.MustMarshalBinaryLengthPrefixed(issuers)
	store.Set([]byte(keyIssuerList), bz)
}

func (k Keeper) AddIssuer(ctx sdk.Context, newIssuer types.Issuer) sdk.Result {
	issuers := k.GetIssuers(ctx)

	existingDenoms := collectDenoms(issuers)
	if anyContained(existingDenoms, newIssuer.Denoms...) {
		return types.ErrDenominationAlreadyAssigned().Result()
	}

	found := false
	for i, _ := range issuers {
		if issuers[i].Address.Equals(newIssuer.Address) {
			issuers[i].Denoms = append(issuers[i].Denoms, newIssuer.Denoms...)
			sort.Strings(issuers[i].Denoms)
			found = true
			break
		}
	}

	if !found {
		issuers = append(issuers, newIssuer)
	}

	k.setIssuers(ctx, issuers)
	k.ik.AddDenoms(ctx, newIssuer.Denoms)
	return sdk.Result{Events: ctx.EventManager().Events()}
}

func (k Keeper) RemoveIssuer(ctx sdk.Context, issuer sdk.AccAddress) sdk.Result {
	issuers := k.GetIssuers(ctx)

	updatedIssuers := make([]types.Issuer, 0)

	// This is one way to remove an element from a slice. There are many. This is one.
	for _, i := range issuers {
		if i.Address.Equals(issuer) {
			continue
		}

		updatedIssuers = append(updatedIssuers, i)
	}

	if len(updatedIssuers) == len(issuers) {
		return types.ErrIssuerNotFound(issuer).Result()
	}

	k.setIssuers(ctx, updatedIssuers)
	return sdk.Result{Events: ctx.EventManager().Events()}
}

func anyContained(s []string, searchterms ...string) bool {
	for _, st := range searchterms {
		index := sort.SearchStrings(s, st)
		if index < len(s) && s[index] == st {
			return true
		}
	}

	return false
}

func collectDenoms(issuers []types.Issuer) (res []string) {
	for _, issuer := range issuers {
		res = append(res, issuer.Denoms...)
	}

	sort.Strings(res)
	return
}

func removeDenom(coins sdk.Coins, denom string) (res sdk.Coins) {
	for _, c := range coins {
		if c.Denom == denom {
			continue
		}

		res = append(res, c)
	}

	return
}

func (k Keeper) mustBeIssuer(ctx sdk.Context, address sdk.AccAddress) (types.Issuer, error) {
	if address == nil {
		return types.Issuer{}, fmt.Errorf("no issuer specified")
	}

	issuers := k.GetIssuers(ctx)

	for _, issuer := range issuers {
		if issuer.Address.Equals(address) {
			return issuer, nil
		}
	}

	k.logger(ctx).Info("Issuer operation attempted by non-issuer", "address", address)
	return types.Issuer{}, fmt.Errorf("%v is not an issuer", address)
}

func (k Keeper) mustBeIssuerOfDenom(ctx sdk.Context, address sdk.AccAddress, denom string) (types.Issuer, error) {
	if address == nil {
		return types.Issuer{}, fmt.Errorf("no issuer specified")
	}

	issuers := k.GetIssuers(ctx)

	for _, issuer := range issuers {
		if issuer.Address.Equals(address) {
			for _, d := range issuer.Denoms {
				if d == denom {
					return issuer, nil
				}
			}
		}
	}

	k.logger(ctx).Info("Issuer operation attempted by non-issuer", "address", address)
	return types.Issuer{}, fmt.Errorf("%v is not an issuer", address)
}

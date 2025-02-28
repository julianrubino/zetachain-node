package keeper

import (
	"context"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	eth "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/zeta-chain/zetacore/x/crosschain/types"
	zetaObserverTypes "github.com/zeta-chain/zetacore/x/observer/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getOutTrackerIndex(chainID int64, nonce uint64) string {
	return fmt.Sprintf("%d-%d", chainID, nonce)
}

// SetOutTxTracker set a specific outTxTracker in the store from its index
func (k Keeper) SetOutTxTracker(ctx sdk.Context, outTxTracker types.OutTxTracker) {
	outTxTracker.Index = getOutTrackerIndex(outTxTracker.ChainId, outTxTracker.Nonce)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.OutTxTrackerKeyPrefix))
	b := k.cdc.MustMarshal(&outTxTracker)
	store.Set(types.OutTxTrackerKey(
		outTxTracker.Index,
	), b)
}

// GetOutTxTracker returns a outTxTracker from its index
func (k Keeper) GetOutTxTracker(
	ctx sdk.Context,
	chainID int64,
	nonce uint64,

) (val types.OutTxTracker, found bool) {
	index := getOutTrackerIndex(chainID, nonce)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.OutTxTrackerKeyPrefix))

	b := store.Get(types.OutTxTrackerKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveOutTxTracker removes a outTxTracker from the store
func (k Keeper) RemoveOutTxTracker(
	ctx sdk.Context,
	chainID int64,
	nonce uint64,

) {
	index := getOutTrackerIndex(chainID, nonce)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.OutTxTrackerKeyPrefix))
	store.Delete(types.OutTxTrackerKey(
		index,
	))
}

// GetAllOutTxTracker returns all outTxTracker
func (k Keeper) GetAllOutTxTracker(ctx sdk.Context) (list []types.OutTxTracker) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.OutTxTrackerKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.OutTxTracker
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// Queries

func (k Keeper) OutTxTrackerAll(c context.Context, req *types.QueryAllOutTxTrackerRequest) (*types.QueryAllOutTxTrackerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var outTxTrackers []types.OutTxTracker
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	outTxTrackerStore := prefix.NewStore(store, types.KeyPrefix(types.OutTxTrackerKeyPrefix))
	pageRes, err := query.Paginate(outTxTrackerStore, req.Pagination, func(key []byte, value []byte) error {
		var outTxTracker types.OutTxTracker
		if err := k.cdc.Unmarshal(value, &outTxTracker); err != nil {
			return err
		}

		outTxTrackers = append(outTxTrackers, outTxTracker)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryAllOutTxTrackerResponse{OutTxTracker: outTxTrackers, Pagination: pageRes}, nil
}

func (k Keeper) OutTxTrackerAllByChain(c context.Context, req *types.QueryAllOutTxTrackerByChainRequest) (*types.QueryAllOutTxTrackerByChainResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var outTxTrackers []types.OutTxTracker
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	outTxTrackerStore := prefix.NewStore(store, types.KeyPrefix(types.OutTxTrackerKeyPrefix))

	pageRes, err := query.Paginate(outTxTrackerStore, req.Pagination, func(key []byte, value []byte) error {
		var outTxTracker types.OutTxTracker
		if err := k.cdc.Unmarshal(value, &outTxTracker); err != nil {
			return err
		}
		if outTxTracker.ChainId == req.Chain {
			outTxTrackers = append(outTxTrackers, outTxTracker)
		}
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllOutTxTrackerByChainResponse{OutTxTracker: outTxTrackers, Pagination: pageRes}, nil
}

func (k Keeper) OutTxTracker(c context.Context, req *types.QueryGetOutTxTrackerRequest) (*types.QueryGetOutTxTrackerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)
	val, found := k.GetOutTxTracker(
		ctx,
		req.ChainID,
		req.Nonce,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetOutTxTrackerResponse{OutTxTracker: val}, nil
}

// Messages

// Adds a new record to the outbound transaction tracker.
//
// Only the admin policy account and the observer validators are authorized to
// broadcast this message.
func (k msgServer) AddToOutTxTracker(goCtx context.Context, msg *types.MsgAddToOutTxTracker) (*types.MsgAddToOutTxTrackerResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	chain := k.zetaObserverKeeper.GetParams(ctx).GetChainFromChainID(msg.ChainId)
	if chain == nil {
		return nil, zetaObserverTypes.ErrSupportedChains
	}

	if msg.Proof == nil { // without proof, only certain accounts can send this message
		adminPolicyAccount := k.zetaObserverKeeper.GetParams(ctx).GetAdminPolicyAccount(zetaObserverTypes.Policy_Type_out_tx_tracker)
		isAdmin := msg.Creator == adminPolicyAccount

		isObserver, err := k.zetaObserverKeeper.IsAuthorized(ctx, msg.Creator, chain)
		if err != nil {
			ctx.Logger().Error("Error while checking if the account is an observer", err)
			return nil, sdkerrors.Wrap(zetaObserverTypes.ErrNotAuthorized, fmt.Sprintf("error  IsAuthorized %s", msg.Creator))
		}
		// Sender needs to be either the admin policy account or an observer
		if !(isAdmin || isObserver) {
			return nil, sdkerrors.Wrap(zetaObserverTypes.ErrNotAuthorized, fmt.Sprintf("Creator %s", msg.Creator))
		}
	}

	proven := false
	if msg.Proof != nil {
		blockHash := eth.HexToHash(msg.BlockHash)
		res, found := k.zetaObserverKeeper.GetBlockHeader(ctx, blockHash.Bytes())
		if !found {
			return nil, sdkerrors.Wrap(zetaObserverTypes.ErrBlockHeaderNotFound, fmt.Sprintf("block header not found %s", msg.BlockHash))
		}
		var header ethtypes.Header
		err := rlp.DecodeBytes(res.Header, &header)
		if err != nil {
			return nil, err
		}
		val, err := msg.Proof.Verify(header.TxHash, int(msg.TxIndex))
		if err == nil {
			var txx ethtypes.Transaction
			err = txx.UnmarshalBinary(val)
			if err != nil {
				return nil, err
			}
			signer := ethtypes.NewLondonSigner(txx.ChainId())
			sender, err := ethtypes.Sender(signer, &txx)
			if err != nil {
				return nil, err
			}
			res, err := k.GetTssAddress(ctx, &types.QueryGetTssAddressRequest{})
			if err != nil {
				return nil, err
			}
			tssAddr := eth.HexToAddress(res.Eth)
			if tssAddr == (eth.Address{}) {
				return nil, fmt.Errorf("tss address not found")
			}
			if sender != tssAddr {
				return nil, fmt.Errorf("sender is not tss address")
			}
			if txx.Nonce() != msg.Nonce {
				return nil, fmt.Errorf("nonce mismatch")
			}
			proven = true
		}

		if !proven {
			return nil, fmt.Errorf("proof failed")
		}
	}

	tracker, found := k.GetOutTxTracker(ctx, msg.ChainId, msg.Nonce)
	hash := types.TxHashList{
		TxHash:   msg.TxHash,
		TxSigner: msg.Creator,
	}
	if !found {
		k.SetOutTxTracker(ctx, types.OutTxTracker{
			Index:    "",
			ChainId:  chain.ChainId,
			Nonce:    msg.Nonce,
			HashList: []*types.TxHashList{&hash},
		})
		return &types.MsgAddToOutTxTrackerResponse{}, nil
	}

	var isDup = false
	for _, hash := range tracker.HashList {
		if strings.EqualFold(hash.TxHash, msg.TxHash) {
			isDup = true
			if proven {
				hash.Proved = true
				k.SetOutTxTracker(ctx, tracker)
				k.Logger(ctx).Info("Proof'd outbound transaction")
				return &types.MsgAddToOutTxTrackerResponse{}, nil
			}
			break
		}
	}
	if !isDup {
		if proven {
			hash.Proved = true
			tracker.HashList = append([]*types.TxHashList{&hash}, tracker.HashList...)
			k.Logger(ctx).Info("Proof'd outbound transaction")
		} else {
			tracker.HashList = append(tracker.HashList, &hash)
		}
		k.SetOutTxTracker(ctx, tracker)
	}
	return &types.MsgAddToOutTxTrackerResponse{}, nil
}

// Removes a record from the outbound transaction tracker by chain ID and nonce.
//
// Only the admin policy account is authorized to broadcast this message.
func (k msgServer) RemoveFromOutTxTracker(goCtx context.Context, msg *types.MsgRemoveFromOutTxTracker) (*types.MsgRemoveFromOutTxTrackerResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if msg.Creator != k.zetaObserverKeeper.GetParams(ctx).GetAdminPolicyAccount(zetaObserverTypes.Policy_Type_out_tx_tracker) {
		return &types.MsgRemoveFromOutTxTrackerResponse{}, zetaObserverTypes.ErrNotAuthorizedPolicy
	}

	k.RemoveOutTxTracker(ctx, msg.ChainId, msg.Nonce)
	return &types.MsgRemoveFromOutTxTrackerResponse{}, nil
}

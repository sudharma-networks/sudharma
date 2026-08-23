package p2p

import (
	"fmt"
	"math/big"
	"time"

	"github.com/sudharma-networks/sudharma/blockchain"
)

// blocksFromHeight returns up to limit blocks beginning
// at startHeight from the node's attached chain.
func (n *Node) blocksFromHeight(
	startHeight uint64,
	limit uint64,
) ([]*blockchain.Block, error) {

	if limit == 0 {
		return nil, fmt.Errorf(
			"block request limit cannot be zero",
		)
	}

	if limit > MaxBlocksPerMessage {
		return nil, fmt.Errorf(
			"block request limit exceeds maximum %d",
			MaxBlocksPerMessage,
		)
	}

	chain := n.Chain()

	if chain == nil {
		return nil, fmt.Errorf(
			"blockchain is not attached",
		)
	}

	chainHeight := chain.Height()

	if startHeight > chainHeight {
		return nil, nil
	}

	blocks := make(
		[]*blockchain.Block,
		0,
		limit,
	)

	for height := startHeight; height <= chainHeight &&
		uint64(len(blocks)) < limit; height++ {

		block, ok :=
			chain.BlockByHeight(
				height,
			)

		if !ok {
			return nil, fmt.Errorf(
				"block %d not found",
				height,
			)
		}

		blocks = append(
			blocks,
			block,
		)
	}

	return blocks, nil
}

// handleGetBlocks responds to a peer asking for
// a consecutive range of blockchain blocks.
func (n *Node) handleGetBlocks(
	peer *PeerConnection,
	message *Message,
) {

	request, err :=
		DecodeGetBlocks(
			message,
		)

	if err != nil {
		fmt.Printf(
			"[SYNC] Invalid block request from %s: %v\n",
			peer.Info.NodeID,
			err,
		)

		return
	}

	blocks, err :=
		n.blocksFromHeight(
			request.StartHeight,
			request.Limit,
		)

	if err != nil {
		fmt.Printf(
			"[SYNC] Failed serving blocks to %s: %v\n",
			peer.Info.NodeID,
			err,
		)

		return
	}

	if len(blocks) == 0 {
		return
	}

	data, err :=
		NewBlocksMessage(
			blocks,
		)

	if err != nil {
		fmt.Printf(
			"[SYNC] Failed encoding blocks for %s: %v\n",
			peer.Info.NodeID,
			err,
		)

		return
	}

	if err :=
		peer.write(
			data,
		); err != nil {

		fmt.Printf(
			"[SYNC] Failed sending blocks to %s: %v\n",
			peer.Info.NodeID,
			err,
		)

		return
	}

	fmt.Printf(
		"[SYNC] Sent %d block(s) to %s starting at height %d\n",
		len(blocks),
		peer.Info.NodeID,
		request.StartHeight,
	)
}

// handleBlocksResponse receives a block batch and
// delivers it to the currently active sync request.
func (n *Node) handleBlocksResponse(
	peer *PeerConnection,
	message *Message,
) {

	blocks, err :=
		DecodeBlocks(
			message,
		)

	if err != nil {
		fmt.Printf(
			"[SYNC] Invalid blocks response from %s: %v\n",
			peer.Info.NodeID,
			err,
		)

		return
	}

	select {
	case n.blocksResponse <- blocks:

	default:
		fmt.Printf(
			"[SYNC] Unexpected blocks response from %s ignored\n",
			peer.Info.NodeID,
		)
	}
}

// RequestBlocks asks one connected peer for blocks.
//
// Only one request can be active at a time in the
// current development implementation.
func (n *Node) RequestBlocks(
	nodeID string,
	startHeight uint64,
	limit uint64,
	timeout time.Duration,
) ([]*blockchain.Block, error) {

	n.syncRequestMu.Lock()
	defer n.syncRequestMu.Unlock()

	if timeout <= 0 {
		return nil, fmt.Errorf(
			"sync timeout must be greater than zero",
		)
	}

	if limit == 0 ||
		limit > MaxBlocksPerMessage {

		return nil, fmt.Errorf(
			"invalid block request limit",
		)
	}

	n.mu.RLock()

	peer, ok :=
		n.peers[nodeID]

	n.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf(
			"peer not found: %s",
			nodeID,
		)
	}

	// Clear any old unused response.
	select {
	case <-n.blocksResponse:
	default:
	}

	data, err :=
		NewGetBlocksMessage(
			startHeight,
			limit,
		)

	if err != nil {
		return nil, err
	}

	if err :=
		peer.write(
			data,
		); err != nil {

		return nil, fmt.Errorf(
			"failed to send block request: %w",
			err,
		)
	}

	timer :=
		time.NewTimer(
			timeout,
		)

	defer timer.Stop()

	select {
	case blocks :=
		<-n.blocksResponse:

		return blocks, nil

	case <-timer.C:
		return nil, fmt.Errorf(
			"timed out waiting for blocks from %s",
			nodeID,
		)
	}
}

// peerChainStatus returns a snapshot of the chain
// information advertised by a connected peer.
func (n *Node) peerChainStatus(
	nodeID string,
) (
	uint64,
	string,
	*big.Int,
	error,
) {

	n.mu.RLock()

	peer, ok :=
		n.peers[nodeID]

	if !ok {
		n.mu.RUnlock()

		return 0, "", nil, fmt.Errorf(
			"peer not found: %s",
			nodeID,
		)
	}

	remoteHeight :=
		peer.Info.Height

	remoteTip :=
		peer.Info.TipHash

	remoteWorkString :=
		peer.Info.TotalWork

	n.mu.RUnlock()

	remoteWork := new(big.Int)

	if _, ok :=
		remoteWork.SetString(
			remoteWorkString,
			10,
		); !ok {

		return 0, "", nil, fmt.Errorf(
			"peer %s advertised invalid total work %q",
			nodeID,
			remoteWorkString,
		)
	}

	if remoteWork.Sign() < 0 {
		return 0, "", nil, fmt.Errorf(
			"peer %s advertised negative total work",
			nodeID,
		)
	}

	return remoteHeight,
		remoteTip,
		remoteWork,
		nil
}

// downloadCandidateChain downloads the peer's complete
// post-genesis chain into an independent candidate.
//
// Nothing in the live blockchain or live state is changed
// while the candidate is being downloaded and validated.
func (n *Node) downloadCandidateChain(
	nodeID string,
	remoteHeight uint64,
	remoteTip string,
	remoteWork *big.Int,
	timeout time.Duration,
) (*blockchain.Chain, error) {

	if remoteWork == nil {
		return nil, fmt.Errorf(
			"remote total work cannot be nil",
		)
	}

	candidate :=
		blockchain.NewChain()

	if remoteHeight == 0 {
		tip := candidate.Tip()

		if tip == nil {
			return nil, fmt.Errorf(
				"candidate genesis tip is nil",
			)
		}

		if tip.Hash() != remoteTip {
			return nil, fmt.Errorf(
				"peer genesis tip does not match Sudharma Network genesis",
			)
		}

		if candidate.TotalWork().Cmp(
			remoteWork,
		) != 0 {

			return nil, fmt.Errorf(
				"peer advertised total work does not match genesis work",
			)
		}

		return candidate, nil
	}

	nextHeight := uint64(1)

	for nextHeight <= remoteHeight {

		remaining :=
			remoteHeight -
				nextHeight +
				1

		limit := remaining

		if limit > MaxBlocksPerMessage {
			limit =
				MaxBlocksPerMessage
		}

		blocks, err :=
			n.RequestBlocks(
				nodeID,
				nextHeight,
				limit,
				timeout,
			)

		if err != nil {
			return nil, fmt.Errorf(
				"failed requesting candidate blocks from height %d: %w",
				nextHeight,
				err,
			)
		}

		if len(blocks) == 0 {
			return nil, fmt.Errorf(
				"peer returned no blocks while candidate download is incomplete",
			)
		}

		for _, block := range blocks {

			if block == nil {
				return nil, fmt.Errorf(
					"peer returned nil block",
				)
			}

			if block.Height !=
				nextHeight {

				return nil, fmt.Errorf(
					"unexpected candidate block height: expected %d, got %d",
					nextHeight,
					block.Height,
				)
			}

			if err :=
				candidate.AddBlock(
					block,
				); err != nil {

				return nil, fmt.Errorf(
					"candidate block %d failed validation: %w",
					block.Height,
					err,
				)
			}

			fmt.Printf(
				"[SYNC] Validated candidate block #%d | %d/%d\n",
				block.Height,
				block.Height,
				remoteHeight,
			)

			nextHeight++

			if nextHeight >
				remoteHeight {

				break
			}
		}
	}

	if candidate.Height() !=
		remoteHeight {

		return nil, fmt.Errorf(
			"candidate ended at wrong height: expected %d, got %d",
			remoteHeight,
			candidate.Height(),
		)
	}

	tip :=
		candidate.Tip()

	if tip == nil {
		return nil, fmt.Errorf(
			"candidate chain tip is nil",
		)
	}

	if tip.Hash() != remoteTip {
		return nil, fmt.Errorf(
			"candidate tip does not match peer advertised tip",
		)
	}

	actualWork :=
		candidate.TotalWork()

	if actualWork.Cmp(
		remoteWork,
	) != 0 {

		return nil, fmt.Errorf(
			"candidate work mismatch: peer advertised %s, validated chain has %s",
			remoteWork.String(),
			actualWork.String(),
		)
	}

	return candidate, nil
}

// removeConfirmedCandidateTransactions removes transactions
// included in the adopted candidate chain from the local mempool.
//
// This helper remains available for other block-processing paths.
// Reorganization now performs a complete mempool rebuild through
// ReorganizeWithMempoolRecovery.
func (n *Node) removeConfirmedCandidateTransactions(
	chain *blockchain.Chain,
) {

	if chain == nil {
		return
	}

	height :=
		chain.Height()

	for blockHeight :=
		uint64(1); blockHeight <= height; blockHeight++ {

		block, ok :=
			chain.BlockByHeight(
				blockHeight,
			)

		if !ok ||
			block == nil {

			continue
		}

		for _, tx := range block.Transactions {

			if tx == nil {
				continue
			}

			n.mempool.RemoveTransaction(
				tx.ID,
			)
		}
	}
}

// SyncFromPeer synchronizes with a connected peer using
// cumulative Proof-of-Work as the primary fork-choice rule.
//
// The peer's advertised total work is never trusted by itself.
// Sudharma Network downloads the candidate chain, validates every block,
// calculates the candidate work locally, rebuilds state by replay,
// and only then may replace the active chain.
//
// When a reorganization occurs, transactions from the abandoned
// fork are reconsidered for the mempool.
func (n *Node) SyncFromPeer(
	nodeID string,
	timeout time.Duration,
) error {

	if timeout <= 0 {
		return fmt.Errorf(
			"sync timeout must be greater than zero",
		)
	}

	chain :=
		n.Chain()

	if chain == nil {
		return fmt.Errorf(
			"blockchain is not attached",
		)
	}

	state :=
		n.State()

	if state == nil {
		return fmt.Errorf(
			"blockchain state is not attached",
		)
	}

	remoteHeight,
		remoteTip,
		remoteWork,
		err :=
		n.peerChainStatus(
			nodeID,
		)

	if err != nil {
		return err
	}

	localTip :=
		chain.Tip()

	if localTip == nil {
		return fmt.Errorf(
			"local chain tip is nil",
		)
	}

	localHeight :=
		chain.Height()

	localWork :=
		chain.TotalWork()

	// Same exact chain tip means there is nothing to do.
	if localHeight == remoteHeight &&
		localTip.Hash() == remoteTip {

		fmt.Printf(
			"[SYNC] Local node already matches peer %s | Height: %d | Work: %s\n",
			nodeID,
			localHeight,
			localWork.String(),
		)

		return nil
	}

	fmt.Printf(
		"[SYNC] Comparing chain with %s | Local: height %d work %s | Remote: height %d work %s\n",
		nodeID,
		localHeight,
		localWork.String(),
		remoteHeight,
		remoteWork.String(),
	)

	// Cumulative Proof-of-Work is the primary fork-choice rule.
	if remoteWork.Cmp(
		localWork,
	) < 0 {

		fmt.Printf(
			"[SYNC] Peer chain has less cumulative work. Keeping local chain.\n",
		)

		return nil
	}

	// Equal work does not justify replacing the current chain
	// unless BetterChain's secondary height rule could prefer it.
	if remoteWork.Cmp(
		localWork,
	) == 0 &&
		remoteHeight <= localHeight {

		fmt.Printf(
			"[SYNC] Peer chain does not beat local chain. Keeping local chain.\n",
		)

		return nil
	}

	fmt.Printf(
		"[SYNC] Peer may have a better chain. Downloading candidate safely...\n",
	)

	candidate, err :=
		n.downloadCandidateChain(
			nodeID,
			remoteHeight,
			remoteTip,
			remoteWork,
			timeout,
		)

	if err != nil {
		return fmt.Errorf(
			"candidate chain download failed: %w",
			err,
		)
	}

	best, err :=
		blockchain.BetterChain(
			chain,
			candidate,
		)

	if err != nil {
		return fmt.Errorf(
			"fork-choice comparison failed: %w",
			err,
		)
	}

	if best == chain {
		fmt.Printf(
			"[SYNC] Validated candidate does not beat local chain. Keeping local chain.\n",
		)

		return nil
	}

	fmt.Printf(
		"[REORG] Better chain validated | Candidate height: %d | Candidate work: %s\n",
		candidate.Height(),
		candidate.TotalWork().String(),
	)

	// =================================================
	// REORGANIZE + RECOVER ORPHANED TRANSACTIONS
	// =================================================

	adopted,
		recovered,
		err :=
		n.ReorganizeWithMempoolRecovery(
			candidate,
		)

	if err != nil {
		return fmt.Errorf(
			"chain reorganization failed: %w",
			err,
		)
	}

	if !adopted {
		fmt.Printf(
			"[REORG] Candidate was not adopted. Local chain remains active.\n",
		)

		return nil
	}

	fmt.Printf(
		"[REORG] Mempool recovery complete | Recovered: %d transaction(s)\n",
		recovered,
	)

	newTip :=
		chain.Tip()

	if newTip == nil {
		return fmt.Errorf(
			"reorganized chain tip is nil",
		)
	}

	fmt.Printf(
		"[REORG] Chain adopted successfully | Height: %d | Work: %s | Tip: %s\n",
		chain.Height(),
		chain.TotalWork().String(),
		newTip.Hash(),
	)

	return nil
}

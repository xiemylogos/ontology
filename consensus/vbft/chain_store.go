/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */

package vbft

import (
	"fmt"

	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/core/ledger"
	"github.com/ontio/ontology/core/store"
	"github.com/ontio/ontology/core/store/overlaydb"
	"github.com/ontio/ontology/events/message"
)

type PendingBlock struct {
	block      *Block
	execResult *store.ExecuteResult
}
type ChainStore struct {
	db              *ledger.Ledger
	chainedBlockNum uint32
	pendingBlocks   map[uint32]*PendingBlock
	server          *Server
	needSubmitBlock bool
}

func OpenBlockStore(db *ledger.Ledger, server *Server) (*ChainStore, error) {
	return &ChainStore{
		db:              db,
		chainedBlockNum: db.GetCurrentBlockHeight(),
		pendingBlocks:   make(map[uint32]*PendingBlock),
		server:          server,
		needSubmitBlock: false,
	}, nil
}

func (self *ChainStore) close() {
	// TODO: any action on ledger actor??
}

func (self *ChainStore) GetChainedBlockNum() uint32 {
	return self.chainedBlockNum
}

func (self *ChainStore) SetExecMerkleRoot(blkNum uint32, merkleRoot common.Uint256) {
	if blk, present := self.pendingBlocks[blkNum]; blk != nil && present {
		blk.execResult.MerkleRoot = merkleRoot
	} else {
		log.Errorf("SetExecMerkleRoot failed blkNum:%d,merkleRoot:%s", blkNum, merkleRoot.ToHexString())
	}
}

func (self *ChainStore) GetExecMerkleRoot(blkNum uint32) common.Uint256 {
	if blk, present := self.pendingBlocks[blkNum]; blk != nil && present {
		return blk.execResult.MerkleRoot
	}
	log.Errorf("GetExecMerkleRoot failed blkNum:%d", blkNum)
	return common.Uint256{}

}

func (self *ChainStore) SetExecWriteSet(blkNum uint32, memdb *overlaydb.MemDB) {
	if blk, present := self.pendingBlocks[blkNum]; blk != nil && present {
		blk.execResult.WriteSet = memdb
	} else {
		log.Errorf("SetExecWriteSet failed blkNum:%d", blkNum)
	}
}

func (self *ChainStore) GetExecWriteSet(blkNum uint32) *overlaydb.MemDB {
	if blk, present := self.pendingBlocks[blkNum]; blk != nil && present {
		return blk.execResult.WriteSet
	}
	log.Errorf("GetExecWriteSet failed blkNum:%d", blkNum)
	return nil
}

func (self *ChainStore) SetExecuteResult(blkNum uint32, execResult *store.ExecuteResult) {
	if blk, present := self.pendingBlocks[blkNum]; blk != nil && present {
		blk.execResult = execResult
	} else {
		log.Errorf("SetExecuteResult failed blkNum:%d", blkNum)
	}
}

func (self *ChainStore) ReloadFromLedger() {
	height := self.db.GetCurrentBlockHeight()
	if height > self.chainedBlockNum {
		// update chainstore height
		self.chainedBlockNum = height
		// remove persisted pending blocks
		newPending := make(map[uint32]*PendingBlock)
		for blkNum, blk := range self.pendingBlocks {
			if blkNum > height {
				newPending[blkNum] = blk
			}
		}
		// update pending blocks
		self.pendingBlocks = newPending
	}
}

func (self *ChainStore) AddBlock(block *PendingBlock) error {
	if block == nil {
		return fmt.Errorf("try add nil block")
	}

	if block.block.getBlockNum() <= self.GetChainedBlockNum() {
		log.Warnf("chain store adding chained block(%d, %d)", block.block.getBlockNum(), self.GetChainedBlockNum())
		return nil
	}

	if block.block.Block.Header == nil {
		panic("nil block header")
	}
	self.pendingBlocks[block.block.getBlockNum()] = block

	blkNum := self.GetChainedBlockNum() + 1
	for {
		if blk, present := self.pendingBlocks[blkNum]; blk != nil && present {
			log.Infof("ledger adding chained block (%d, %d)", blkNum, self.GetChainedBlockNum())

			var err error
			if self.needSubmitBlock {
				if submitBlk, present := self.pendingBlocks[blkNum-1]; submitBlk != nil && present {
					err := self.db.SubmitBlock(submitBlk.block.Block, *submitBlk.execResult)
					if err != nil && blkNum > self.GetChainedBlockNum() {
						return fmt.Errorf("ledger add submitBlk (%d, %d) failed: %s", blkNum, self.GetChainedBlockNum(), err)
					}
					if _, present := self.pendingBlocks[blkNum-2]; present {
						delete(self.pendingBlocks, blkNum-2)
					}
				} else {
					break
				}
			}
			execResult, err := self.db.ExecuteBlock(blk.block.Block)
			if err != nil {
				log.Errorf("chainstore AddBlock GetBlockExecResult: %s", err)
				return fmt.Errorf("GetBlockExecResult: %s", err)
			}
			self.SetExecuteResult(blk.block.getBlockNum(), &execResult)
			self.needSubmitBlock = true
			self.server.pid.Tell(
				&message.BlockConsensusComplete{
					Block: blk.block.Block,
				})
			self.chainedBlockNum = blkNum
			/*
				if blkNum != self.db.GetCurrentBlockHeight() {
					log.Errorf("!!! chain store added chained block (%d, %d): %s",
						blkNum, self.db.GetCurrentBlockHeight(), err)
				}
			*/
			blkNum++
		} else {
			break
		}
	}

	return nil
}

func (self *ChainStore) SetBlock(blkNum uint32, blk *PendingBlock) {
	self.pendingBlocks[blkNum] = blk
}

func (self *ChainStore) GetBlock(blockNum uint32) (*PendingBlock, error) {

	if blk, present := self.pendingBlocks[blockNum]; present {
		return blk, nil
	}

	block, err := self.db.GetBlockByHeight(uint32(blockNum))
	if err != nil {
		return nil, err
	}
	merkleRoot, err := self.db.GetStateMerkleRoot(blockNum)
	if err != nil {
		log.Errorf("GetStateMerkleRoot blockNum:%d, error :%s", blockNum, err)
		return nil, fmt.Errorf("GetStateMerkleRoot blockNum:%d, error :%s", blockNum, err)
	}
	blockInfo, err := initVbftBlock(block, merkleRoot)
	if err != nil {
		return nil, err
	}
	pendingBlock := &PendingBlock{
		block: blockInfo,
		execResult: &store.ExecuteResult{
			MerkleRoot: merkleRoot,
		},
	}
	return pendingBlock, nil
	//return initVbftBlock(block, merkleRoot)
}

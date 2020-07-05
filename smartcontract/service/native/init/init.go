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

package init

import (
	"bytes"
	"math/big"

	"github.com/xiemylogos/ontology/v2/common"
	"github.com/xiemylogos/ontology/v2/smartcontract/service/native/auth"
	"github.com/xiemylogos/ontology/v2/smartcontract/service/native/cross_chain/cross_chain_manager"
	"github.com/xiemylogos/ontology/v2/smartcontract/service/native/cross_chain/header_sync"
	"github.com/xiemylogos/ontology/v2/smartcontract/service/native/cross_chain/lock_proxy"
	params "github.com/xiemylogos/ontology/v2/smartcontract/service/native/global_params"
	"github.com/xiemylogos/ontology/v2/smartcontract/service/native/governance"
	"github.com/xiemylogos/ontology/v2/smartcontract/service/native/ong"
	"github.com/xiemylogos/ontology/v2/smartcontract/service/native/ont"
	"github.com/xiemylogos/ontology/v2/smartcontract/service/native/ontfs"
	"github.com/xiemylogos/ontology/v2/smartcontract/service/native/ontid"
	"github.com/xiemylogos/ontology/v2/smartcontract/service/native/utils"
	"github.com/xiemylogos/ontology/v2/smartcontract/service/neovm"
	vm "github.com/xiemylogos/ontology/v2/vm/neovm"
)

var (
	COMMIT_DPOS_BYTES = InitBytes(utils.GovernanceContractAddress, governance.COMMIT_DPOS)
)

func init() {
	ong.InitOng()
	ont.InitOnt()
	params.InitGlobalParams()
	ontid.Init()
	auth.Init()
	governance.InitGovernance()
	cross_chain_manager.InitCrossChain()
	header_sync.InitHeaderSync()
	lock_proxy.InitLockProxy()
	ontfs.InitFs()
}

func InitBytes(addr common.Address, method string) []byte {
	bf := new(bytes.Buffer)
	builder := vm.NewParamsBuilder(bf)
	builder.EmitPushByteArray([]byte{})
	builder.EmitPushByteArray([]byte(method))
	builder.EmitPushByteArray(addr[:])
	builder.EmitPushInteger(big.NewInt(0))
	builder.Emit(vm.SYSCALL)
	builder.EmitPushByteArray([]byte(neovm.NATIVE_INVOKE_NAME))

	return builder.ToArray()
}

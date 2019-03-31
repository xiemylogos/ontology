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

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/gosuri/uiprogress"
	"github.com/ontio/ontology/cmd/utils"
	"github.com/ontio/ontology/common/config"
	"github.com/ontio/ontology/common/serialization"
	"github.com/ontio/ontology/core/ledger"
	"github.com/urfave/cli"
)

var ExportCommand = cli.Command{
	Name:      "export",
	Usage:     "Export blocks in DB to a file",
	ArgsUsage: "",
	Action:    exportBlocks,
	Flags: []cli.Flag{
		utils.ExportFileFlag,
		utils.ExportStartHeightFlag,
		utils.ExportEndHeightFlag,
		utils.ExportSpeedFlag,
		utils.ExportDisableSignFlag,
	},
	Description: "",
}

func exportBlocks(ctx *cli.Context) error {
	cfg, err := SetOntologyConfig(ctx)
	if err != nil {
		PrintErrorMsg("SetOntologyConfig error:%s", err)
		cli.ShowSubcommandHelp(ctx)
		return nil
	}
	exportFile := ctx.String(utils.GetFlagName(utils.ExportFileFlag))
	if exportFile == "" {
		PrintErrorMsg("Missing %s argument.", utils.ExportFileFlag.Name)
		cli.ShowSubcommandHelp(ctx)
		return nil
	}

	startHeight := ctx.Uint(utils.GetFlagName(utils.ExportStartHeightFlag))
	endHeight := ctx.Uint(utils.GetFlagName(utils.ExportEndHeightFlag))
	if endHeight > 0 && startHeight > endHeight {
		return fmt.Errorf("export error: start height should smaller than end height")
	}
	blockCount, err := utils.GetBlockCount()
	if err != nil {
		return fmt.Errorf("GetBlockCount error:%s", err)
	}
	currentBlockHeight := uint(blockCount - 1)
	if startHeight > currentBlockHeight {
		PrintWarnMsg("StartBlockHeight:%d larger than CurrentBlockHeight:%d, No blocks to export.", startHeight, currentBlockHeight)
		return nil
	}
	if endHeight == 0 || endHeight > currentBlockHeight {
		endHeight = currentBlockHeight
	}

	speed := ctx.String(utils.GetFlagName(utils.ExportSpeedFlag))
	var sleepTime time.Duration
	switch speed {
	case "h":
		sleepTime = 0
	case "m":
		sleepTime = time.Millisecond * 2
	default:
		sleepTime = time.Millisecond * 5
	}

	exportFile = utils.GenExportBlocksFileName(exportFile, uint32(startHeight), uint32(endHeight))
	ef, err := os.OpenFile(exportFile, os.O_RDWR|os.O_CREATE, 0664)
	if err != nil {
		return fmt.Errorf("open file:%s error:%s", exportFile, err)
	}
	defer ef.Close()
	fWriter := bufio.NewWriter(ef)

	metadata := utils.NewExportBlockMetadata()
	metadata.StartBlockHeight = uint32(startHeight)
	metadata.EndBlockHeight = uint32(endHeight)
	err = metadata.Serialize(fWriter)
	if err != nil {
		return fmt.Errorf("write export metadata error:%s", err)
	}

	//progress bar
	uiprogress.Start()
	bar := uiprogress.AddBar(int(endHeight - startHeight + 1)).
		AppendCompleted().
		AppendElapsed().
		PrependFunc(func(b *uiprogress.Bar) string {
			return fmt.Sprintf("Block(%d/%d)", b.Current()+int(startHeight), int(endHeight))
		})

	PrintInfoMsg("Start export.")
	dbDir := utils.GetStoreDirPath(config.DefConfig.Common.DataDir, config.DefConfig.P2PNode.NetworkName)

	stateHashHeight := config.GetStateHashCheckHeight(cfg.P2PNode.NetworkId)
	ledger.DefLedger, err = ledger.NewLedger(dbDir, stateHashHeight)
	if err != nil {
		return fmt.Errorf("NewLedger error:%s", err)
	}
	for i := uint32(startHeight); i <= uint32(endHeight); i++ {
		block, err := ledger.DefLedger.GetBlockByHeight(i)
		if err != nil {
			return fmt.Errorf("GetBlock:%d error:%s", i, err)
		}
		if ctx.Bool(utils.GetFlagName(utils.ExportDisableSignFlag)) {
			block.Header.SigData = [][]byte{}
		}
		blockData := block.ToArray()
		data, err := utils.CompressBlockData(blockData, metadata.CompressType)
		if err != nil {
			return fmt.Errorf("CompressBlockData height:%d error:%s", i, err)
		}
		err = serialization.WriteUint32(fWriter, uint32(len(data)))
		if err != nil {
			return fmt.Errorf("write block data height:%d len:%d error:%s", i, uint32(len(data)), err)
		}
		_, err = fWriter.Write(data)
		if err != nil {
			return fmt.Errorf("write block data height:%d error:%s", i, err)
		}
		if sleepTime > 0 {
			time.Sleep(sleepTime)
		}
		bar.Incr()
	}
	uiprogress.Stop()

	err = fWriter.Flush()
	if err != nil {
		return fmt.Errorf("export flush file error:%s", err)
	}
	PrintInfoMsg("Export blocks successfully.")
	PrintInfoMsg("StartBlockHeight:%d", startHeight)
	PrintInfoMsg("EndBlockHeight:%d", endHeight)
	PrintInfoMsg("Export file:%s", exportFile)
	return nil
}

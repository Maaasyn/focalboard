package app

import (
	"github.com/mattermost/focalboard/server/model"
	"github.com/mattermost/focalboard/server/services/notify"

	"github.com/mattermost/mattermost-server/v6/shared/mlog"
)

func (a *App) CreateBoardsAndBlocks(bab *model.BoardsAndBlocks, userID string, addMember bool) (*model.BoardsAndBlocks, error) {
	var newBab *model.BoardsAndBlocks
	var members []*model.BoardMember
	var err error

	if addMember {
		newBab, members, err = a.store.CreateBoardsAndBlocksWithAdmin(bab, userID)
	} else {
		newBab, err = a.store.CreateBoardsAndBlocks(bab, userID)
	}

	if err != nil {
		return nil, err
	}

	// all new boards should belong to the same team
	teamID := newBab.Boards[0].TeamID

	go func() {
		for _, board := range newBab.Boards {
			a.wsAdapter.BroadcastBoardChange(teamID, board)
		}

		for _, block := range newBab.Blocks {
			a.wsAdapter.BroadcastBlockChange(teamID, block)
			a.metrics.IncrementBlocksInserted(1)
			a.webhook.NotifyUpdate(block)
			a.notifyBlockChanged(notify.Add, &block, nil, userID)
		}

		if addMember {
			for _, member := range members {
				a.wsAdapter.BroadcastMemberChange(teamID, member.BoardID, member)
			}
		}
	}()

	return newBab, nil
}

func (a *App) PatchBoardsAndBlocks(pbab *model.PatchBoardsAndBlocks, userID string) (*model.BoardsAndBlocks, error) {
	oldBlocksMap := map[string]*model.Block{}
	for _, blockID := range pbab.BlockIDs {
		block, err := a.store.GetBlock(blockID)
		if err != nil {
			return nil, err
		}
		oldBlocksMap[blockID] = block
	}

	bab, err := a.store.PatchBoardsAndBlocks(pbab, userID)
	if err != nil {
		return nil, err
	}

	go func() {
		teamID := bab.Boards[0].TeamID

		for _, block := range bab.Blocks {
			oldBlock, ok := oldBlocksMap[block.ID]
			if !ok {
				a.logger.Error("Error notifying for block change on patch boards and blocks; cannot get old block", mlog.String("blockID", block.ID))
				continue
			}

			a.metrics.IncrementBlocksPatched(1)
			a.wsAdapter.BroadcastBlockChange(teamID, block)
			a.webhook.NotifyUpdate(block)
			a.notifyBlockChanged(notify.Update, &block, oldBlock, userID)
		}

		for _, board := range bab.Boards {
			a.wsAdapter.BroadcastBoardChange(board.TeamID, board)
		}
	}()

	return bab, nil
}

func (a *App) DeleteBoardsAndBlocks(dbab *model.DeleteBoardsAndBlocks, userID string) error {
	firstBoard, err := a.store.GetBoard(dbab.Boards[0])
	if err != nil {
		return err
	}

	// we need the block entity to notify of the block changes, so we
	// fetch and store the blocks first
	blocks := []*model.Block{}
	for _, blockID := range dbab.Blocks {
		block, err := a.store.GetBlock(blockID)
		if err != nil {
			return err
		}
		blocks = append(blocks, block)
	}

	if err := a.store.DeleteBoardsAndBlocks(dbab, userID); err != nil {
		return err
	}

	go func() {
		for _, block := range blocks {
			a.wsAdapter.BroadcastBlockDelete(firstBoard.TeamID, block.ID, block.BoardID)
			a.metrics.IncrementBlocksDeleted(1)
			a.notifyBlockChanged(notify.Update, block, block, userID)
		}

		for _, boardID := range dbab.Boards {
			a.wsAdapter.BroadcastBoardDelete(firstBoard.TeamID, boardID)
		}
	}()

	return nil
}

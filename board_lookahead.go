package main

import (
	"math"

	"github.com/apex/log"
	"gorm.io/gorm"
)

func (board *Board) lookahead3(isPurple bool) error {
	if len(board.Children) == 0 {
		if err := board.lookahead(isPurple); err != nil {
			return err
		}
	}
	for _, child := range board.Children {
		child, err := getBoard(child.ID)
		if err != nil {
			log.WithError(err).Fatal("error")
		}
		if err := child.lookahead2(!isPurple); err != nil {
			return err
		}
	}
	return board.lookahead(isPurple)
}

func (board *Board) lookahead2(isPurple bool) error {
	if len(board.Children) == 0 {
		if err := board.lookahead(isPurple); err != nil {
			return err
		}
	}
	for _, child := range board.Children {
		child, err := getBoard(child.ID)
		if err != nil {
			log.WithError(err).Fatal("error")
		}
		if len(child.Children) == 0 {
			if err := child.lookahead(!isPurple); err != nil {
				return err
			}
		}
	}
	return board.lookahead(isPurple)
}

func (board *Board) lookahead(isPurple bool) error {
	if board.end() {
		board.ActiveCheckMate = true
		board.ActiveScore = -2
		board.InactiveScore = 3
		return db.Save(board).Error
	}
	activeCheck := board.ActiveCheck
	states := board.lookaheadBoards(isPurple)
	board.Children = make([]Board, 0, 32)
	for state := range states {
		state = state.swap()
		b, err := makeBoard(state)
		if err != nil {
			return err
		}
		if activeCheck && (b.InactiveCheck || b.InactiveCheckMate) {
			continue
		}
		board.Children = append(board.Children, b)
	}
	if len(board.Children) == 0 {
		board.ActiveCheckMate = true
		board.ActiveScore = -2
		board.InactiveScore = 3
		return db.Save(board).Error
	}
	activeScore := 0
	inactiveScore := 0
	moves := uint(math.MaxUint64)
	activeCheck = false
	activeCheckMate := true
	for _, child := range board.Children {
		activeScore = activeScore + child.InactiveScore
		inactiveScore = inactiveScore + child.ActiveScore
		if child.Moves < moves {
			moves = child.Moves
		}
		if !activeCheck {
			activeCheck = child.InactiveCheck
		}
		if activeCheckMate {
			activeCheckMate = child.InactiveCheckMate
		}
	}
	board.ActiveCheck = activeCheck
	board.ActiveCheckMate = activeCheckMate
	board.ActiveScore = activeScore
	board.InactiveScore = inactiveScore
	board.Moves = moves + 1
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM game_play WHERE board_id = ?", board.ID).Error; err != nil {
			return err
		}
		return tx.Save(board).Error
	})
}

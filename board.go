package main

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Board board.
type Board struct {
	gorm.Model

	ActiveCheck       bool
	ActiveCheckMate   bool
	ActiveScore       int
	Board             chessState `gorm:"<-:create;type:varchar;size:136;uniqueIndex;not null"`
	Children          []Board    `gorm:"many2many:game_play"`
	InactiveCheck     bool
	InactiveCheckMate bool
	InactiveScore     int
	Moves             uint
}

type move struct {
	piece     rune
	depart    int
	capture   bool
	dest      int
	promotion rune

	castling byte
}

type chessState [64]uint8

func (board chessState) makeMoveBase(isPurple bool, piece uint8, start int) move {
	var p rune
	if isPurple {
		p = valueToPiecePurple[piece&0xE]
	} else {
		p = valueToPieceGreen[piece&0xE]
	}
	return move{piece: p, depart: start}
}

func (board chessState) makeMove(isPurple bool, piece uint8, start, end int) move {
	var p rune
	if isPurple {
		p = valueToPiecePurple[piece&0xE]
	} else {
		p = valueToPieceGreen[piece&0xE]
	}
	return move{piece: p, depart: start, capture: inactivePiece(board[end]), dest: end}
}

func (board chessState) makeMoveCastle(isPurple bool, piece uint8, start int, castling byte) move {
	var p rune
	if isPurple {
		p = valueToPiecePurple[piece&0xE]
	} else {
		p = valueToPieceGreen[piece&0xE]
	}
	return move{piece: p, depart: start, castling: castling}
}

func (board chessState) makeMovePromotion(isPurple bool, piece uint8, start, end int, promotion uint8) move {
	var p rune
	var prom rune
	if isPurple {
		p = valueToPiecePurple[piece&0xE]
		prom = valueToPiecePurple[promotion&0xE]
	} else {
		p = valueToPieceGreen[piece&0xE]
		prom = valueToPieceGreen[promotion&0xE]
	}
	return move{piece: p, depart: start, capture: inactivePiece(board[end]), dest: end, promotion: prom}
}

func makeBoard(state chessState) (Board, error) {
	var board Board
	if err := db.FirstOrCreate(&board, Board{Board: state}).Error; err != nil {
		return Board{}, err
	}
	return board, nil
}

func getBoard(id uint) (Board, error) {
	var board Board
	if err := db.Preload(clause.Associations).First(&board, id).Error; err != nil {
		return Board{}, err
	}
	return board, nil
}

func getBoardByBoard(state chessState) (Board, error) {
	var board Board
	if err := db.Preload(clause.Associations).Where(Board{Board: state}).First(&board).Error; err != nil {
		return Board{}, err
	}
	return board, nil
}

func (board chessState) swap() chessState {
	var state chessState
	copy(state[:], board[:])
	for i, piece := range state {
		if piece != 0 {
			state[i] = piece ^ 1
		}
	}
	return state
}

func (board chessState) contains(piece uint8) bool {
	for _, p := range board {
		if p&0xF == piece {
			return true
		}
	}
	return false
}

func (board chessState) hasKings() bool {
	return board.contains(king|1) && board.contains(king)
}

func (board Board) end() bool {
	return (!board.Board.hasKings()) || board.ActiveCheckMate || board.InactiveCheckMate
}

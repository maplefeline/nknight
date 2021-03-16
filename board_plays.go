package main

import (
	"sync"

	"github.com/apex/log"
)

func (board chessState) validateMove(start, end int) bool {
	if activePiece(board[end]) {
		return false
	}
	shift := end - start
	if shift == 9 || shift == 8 || shift == 7 || shift == 1 || shift == -1 || shift == -7 || shift == -8 || shift == -9 {
		return true
	}
	for _, mod := range []int{7, 8, 9} {
		if shift%mod == 0 {
			if shift < 0 {
				shift = mod
			} else {
				shift = -mod
			}
			break
		}
	}
	if board[end+shift] != 0 {
		return false
	}
	return true
}

func (board chessState) moveForBasic(moves chan move, isPurple bool, piece uint8, start, end int) bool {
	m := board.makeMove(isPurple, piece, start, end)
	if board.validateMove(start, end) {
		moves <- m
		return true
	}
	return false
}

func (board chessState) movesForBishop(moves chan move, isPurple bool, piece uint8, start int) {
	for end := start + 9; end < 64; end = end + 9 {
		if !board.moveForBasic(moves, isPurple, piece, start, end) {
			break
		}
	}
	for end := start + 7; end < 63; end = end + 7 {
		if !board.moveForBasic(moves, isPurple, piece, start, end) {
			break
		}
	}
	for end := start - 7; end >= 1; end = end - 7 {
		if !board.moveForBasic(moves, isPurple, piece, start, end) {
			break
		}
	}
	for end := start - 9; end >= 0; end = end - 9 {
		if !board.moveForBasic(moves, isPurple, piece, start, end) {
			break
		}
	}
}

func (board chessState) movesForKing(moves chan move, isPurple bool, piece uint8, start int) {
	for _, shift := range []int{9, 8, 7, 1, -1, -7, -8, -9} {
		end := start + shift
		if (!(end < 64 && end >= 0)) || activePiece(board[end]) {
			continue
		}
		moves <- board.makeMove(isPurple, piece, start, end)
	}
	if piece&0x10 == 0 {
		return
	}
	if isPurple {
		if board[56] == rook&0x11 && board[57] == 0 && board[58] == 0 && board[59] == 0 {
			moves <- board.makeMoveCastle(isPurple, piece, start, 'q')
		}
		if board[63] == rook&0x11 && board[62] == 0 && board[61] == 0 {
			moves <- board.makeMoveCastle(isPurple, piece, start, 'k')
		}
	} else {
		if board[0] == rook&0x11 && board[1] == 0 && board[2] == 0 && board[3] == 0 {
			moves <- board.makeMoveCastle(isPurple, piece, start, 'q')
		}
		if board[7] == rook&0x11 && board[6] == 0 && board[5] == 0 {
			moves <- board.makeMoveCastle(isPurple, piece, start, 'k')
		}
	}
}

func (board chessState) movesForKnight(moves chan move, isPurple bool, piece uint8, start int) {
	for _, shift := range []int{17, 15, 10, 6, -6, -10, -15, -17} {
		if (start < 8 && shift == -6) || (start > 55 && shift == 6) {
			continue
		}
		end := start + shift
		if (!(end < 64 && end >= 0)) || activePiece(board[end]) {
			continue
		}
		moves <- board.makeMove(isPurple, piece, start, end)
	}
}

func (board chessState) promotionForPawn(moves chan move, isPurple bool, piece uint8, start, end int) {
	_, destRank := rankAndFile(end)
	if (isPurple && destRank < 8) || (!isPurple && destRank > 1) {
		moves <- board.makeMove(isPurple, piece, start, end)
		return
	}
	for _, promotion := range []uint8{bishop, knight, queen, rook} {
		moves <- board.makeMovePromotion(isPurple, piece, start, end, promotion)
	}
}

func (board chessState) movesForPawn(moves chan move, isPurple bool, piece uint8, start int) {
	departFile, departRank := rankAndFile(start)
	if isPurple {
		if departRank == 2 && board[start+8] == 0 && board[start+16] == 0 {
			moves <- board.makeMove(isPurple, piece, start, start+16)
		}
		if board[start+8] == 0 {
			board.promotionForPawn(moves, isPurple, piece, start, start+8)
		}
		if departFile > 'a' && inactivePiece(board[start+9]) {
			board.promotionForPawn(moves, isPurple, piece, start, start+9)
		}
		if departFile < 'h' && inactivePiece(board[start+7]) {
			board.promotionForPawn(moves, isPurple, piece, start, start+7)
		}
	} else {
		if departRank == 7 && board[start-8] == 0 && board[start-16] == 0 {
			moves <- board.makeMove(isPurple, piece, start, start-16)
		}
		if board[start-8] == 0 {
			board.promotionForPawn(moves, isPurple, piece, start, start-8)
		}
		if departFile > 'a' && inactivePiece(board[start-9]) {
			board.promotionForPawn(moves, isPurple, piece, start, start-9)
		}
		if departFile < 'h' && inactivePiece(board[start-7]) {
			board.promotionForPawn(moves, isPurple, piece, start, start-7)
		}
	}
}

func (board chessState) movesForQueen(moves chan move, isPurple bool, piece uint8, start int) {
	board.movesForBishop(moves, isPurple, piece, start)
	board.movesForRook(moves, isPurple, piece, start)
}

func (board chessState) movesForRook(moves chan move, isPurple bool, piece uint8, start int) {
	_, departRank := rankAndFile(start)
	for end := start + 8; end < 64; end = end + 8 {
		if !board.moveForBasic(moves, isPurple, piece, start, end) {
			break
		}
	}
	for end := start + 1; end < int(departRank*8); end = end + 1 {
		if !board.moveForBasic(moves, isPurple, piece, start, end) {
			break
		}
	}
	for end := start - 1; end > int(departRank-1)*8; end = end - 1 {
		if !board.moveForBasic(moves, isPurple, piece, start, end) {
			break
		}
	}
	for end := start - 8; end >= 0; end = end - 8 {
		if !board.moveForBasic(moves, isPurple, piece, start, end) {
			break
		}
	}
}

func (board chessState) movesForPiece(group *sync.WaitGroup, moves chan move, isPurple bool, piece uint8, start int) {
	group.Add(1)
	go func() {
		defer group.Done()
		switch piece & 0xE {
		case bishop:
			board.movesForBishop(moves, isPurple, piece, start)
		case king:
			board.movesForKing(moves, isPurple, piece, start)
		case knight:
			board.movesForKnight(moves, isPurple, piece, start)
		case pawn:
			board.movesForPawn(moves, isPurple, piece, start)
		case queen:
			board.movesForQueen(moves, isPurple, piece, start)
		case rook:
			board.movesForRook(moves, isPurple, piece, start)
		default:
			log.Fatal("invalid piece")
		}
	}()
}

func (board chessState) movesForBoard(isPurple bool) <-chan move {
	if !board.hasKings() {
		return nil
	}
	moves := make(chan move, 32)
	go func() {
		defer close(moves)
		var group sync.WaitGroup
		for start, piece := range board {
			if activePiece(piece) {
				board.movesForPiece(&group, moves, isPurple, piece, start)
			}
		}
		group.Wait()
	}()
	return moves
}

func activePiece(piece uint8) bool {
	return piece&1 != 0 && piece&0xE != 0
}

func inactivePiece(piece uint8) bool {
	return piece&1 == 0 && piece&0xE != 0
}

func (board chessState) movesToBoards(moves <-chan move, isPurple bool) <-chan chessState {
	boards := make(chan chessState, 32)
	go func() {
		defer close(boards)
		for move := range moves {
			var state chessState
			copy(state[:], board[:])
			state[move.depart] = 0
			if move.castling == 'q' {
				state[0] = 0
				state[1] = king | 1
				state[2] = rook | 1
				boards <- state
				var state2 chessState
				copy(state2[:], state[:])
				state2[1] = 0
				state2[2] = king | 1
				state2[3] = rook | 1
				boards <- state2
			} else if move.castling == 'k' {
				state[7] = 0
				state[6] = king | 1
				state[5] = rook | 1
				boards <- state
			} else if move.promotion != rune(0) {
				if isPurple {
					state[move.dest] = pieceToValuePurple[move.promotion] | 1
				} else {
					state[move.dest] = pieceToValueGreen[move.promotion] | 1
				}
				boards <- state
			} else {
				if isPurple {
					state[move.dest] = pieceToValuePurple[move.piece] | 1
				} else {
					state[move.dest] = pieceToValueGreen[move.piece] | 1
				}
				boards <- state
			}
		}
	}()
	return boards
}

func (board Board) lookaheadBoards(isPurple bool) <-chan chessState {
	if !board.InactiveCheckMate {
		check := false
		for state := range board.Board.movesToBoards(board.Board.movesForBoard(isPurple), isPurple) {
			if !check {
				check = !state.hasKings()
			}
		}
		board.InactiveCheckMate = check
	}
	boards := make(chan chessState, 32)
	go func() {
		defer close(boards)
		for state := range board.Board.movesToBoards(board.Board.movesForBoard(isPurple), isPurple) {
			if board.InactiveCheckMate {
				if !state.hasKings() {
					boards <- state
				}
			} else {
				boards <- state
			}
		}
	}()
	return boards
}

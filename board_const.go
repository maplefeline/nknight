package main

const (
	zero   uint8 = iota << 1
	bishop uint8 = iota << 1
	king   uint8 = iota << 1
	knight uint8 = iota << 1
	pawn   uint8 = iota << 1
	queen  uint8 = iota << 1
	rook   uint8 = iota << 1
)

var initialBoard = chessState{
	rook, knight, bishop, queen, king, bishop, knight, rook,
	pawn, pawn, pawn, pawn, pawn, pawn, pawn, pawn,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	pawn | 1, pawn | 1, pawn | 1, pawn | 1, pawn | 1, pawn | 1, pawn | 1, pawn | 1,
	rook | 1, knight | 1, bishop | 1, queen | 1, king | 1, bishop | 1, knight | 1, rook | 1,
}

var valueToPieceGreen = map[uint8]rune{
	bishop: '♝',
	king:   '♚',
	knight: '♞',
	pawn:   '♟',
	queen:  '♛',
	rook:   '♜',
}
var valueToPiecePurple = map[uint8]rune{
	bishop: '♗',
	king:   '♔',
	knight: '♘',
	pawn:   '♙',
	queen:  '♕',
	rook:   '♖',
}
var pieceToValueGreen = map[rune]uint8{
	'♝': bishop,
	'♚': king,
	'♞': knight,
	'♟': pawn,
	'♛': queen,
	'♜': rook,
}
var pieceToValuePurple = map[rune]uint8{
	'♗': bishop,
	'♔': king,
	'♘': knight,
	'♙': pawn,
	'♕': queen,
	'♖': rook,
}

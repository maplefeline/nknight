package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	uuid "github.com/satori/go.uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Game game.
type Game struct {
	gorm.Model

	ActiveAgent       uuid.UUID `gorm:"type:varchar;size:20;index"`
	ActiveAgentPurple bool
	ActiveAgentType   string
	BoardID           uint
	Board             Board
	End               bool
	GameID            uuid.UUID `gorm:"<-:create;type:varchar;size:20;uniqueIndex"`
	InactiveAgent     uuid.UUID `gorm:"type:varchar;size:20;index"`
	InactiveAgentType string
	MoveCount         int
	MovesSincePawn    int
}

func gameIdle() error {
	var count int64
	if err := db.Model(&Game{}).Where(Game{InactiveAgent: placeHolder}).Count(&count).Error; err != nil {
		return err
	}
	if count < 10 {
		for i := 0; i < 3; i++ {
			game, err := makeGame()
			if err != nil {
				return err
			}
			if _, err := game.makeAgent("agent"); err != nil {
				return err
			}
		}
	}
	return db.Where(Game{End: true}).Not(Game{ActiveAgentType: "user"}).Not(Game{InactiveAgentType: "user"}).Delete(&Game{}).Error
}

func makeGame() (*Game, error) {
	board, err := getBoardByBoard(initialBoard)
	if err != nil {
		return nil, err
	}
	id := uuid.NewV4()
	if err := db.Create(&Game{GameID: id, Board: board, ActiveAgent: placeHolder, ActiveAgentPurple: true, InactiveAgent: placeHolder}).Error; err != nil {
		return nil, err
	}
	return getGame(id)
}

func getGame(id uuid.UUID) (*Game, error) {
	var game Game
	if err := db.Preload(clause.Associations).First(&game, Game{GameID: id}).Error; err != nil {
		return nil, err
	}
	return &game, nil
}

func getGames() ([]Game, error) {
	var games []Game
	if err := db.Where(Game{InactiveAgent: placeHolder}).Find(&games).Error; err != nil {
		return nil, err
	}
	for _, game := range games {
		game.ActiveAgent = uuid.Nil
	}
	return games, nil
}

func (game Game) response(agentID uuid.UUID) Game {
	if !game.End {
		if !uuid.Equal(game.ActiveAgent, agentID) {
			game.ActiveAgent = uuid.Nil
		}
		if !uuid.Equal(game.InactiveAgent, agentID) {
			game.InactiveAgent = uuid.Nil
		}
	}
	return game
}

func (game *Game) addAgent(id uuid.UUID, agentType string) error {
	if !uuid.Equal(placeHolder, game.InactiveAgent) {
		return echo.NewHTTPError(http.StatusBadRequest, "game is full")
	}
	if uuid.Equal(placeHolder, game.ActiveAgent) {
		game.ActiveAgent = id
		game.ActiveAgentType = agentType
	} else {
		game.InactiveAgent = id
		game.InactiveAgentType = agentType
	}
	if err := db.Save(&game).Error; err != nil {
		return err
	}
	if !uuid.Equal(placeHolder, game.InactiveAgent) {
		return game.pokeAgent()
	}
	return nil
}

func (game *Game) pokeAgent() error {
	if game.ActiveAgentType != "user" {
		return game.playRound(game.ActiveAgent, nil)
	}
	return nil
}

func (game Game) getPlays() ([]chessState, []move, error) {
	board, err := getBoard(game.BoardID)
	if err != nil {
		return nil, nil, err
	}
	boards := make([]chessState, 0, len(board.Children))
	for _, child := range board.Children {
		boards = append(boards, child.Board)
	}
	moves := make([]move, 0, len(boards))
	for move := range game.Board.Board.movesForBoard(game.ActiveAgentPurple) {
		moves = append(moves, move)
	}
	return boards, moves, nil
}

func (game Game) moveToBoard(m move) *chessState {
	moves := make(chan move)
	moves <- m
	close(moves)
	var board chessState
	for board = range game.Board.Board.movesToBoards(moves, game.ActiveAgentPurple) {
	}
	return &board
}

func (game *Game) validMove(state chessState) (bool, error) {
	board, err := getBoard(game.BoardID)
	if err != nil {
		return false, err
	}
	child, err := getBoardByBoard(state)
	if err != nil {
		return false, err
	}
	var count int64
	if err := db.Table("game_play").Where("board_id", board.ID).Where("child_id", child.ID).Count(&count).Error; err != nil {
		return false, err
	}
	if count != 1 {
		return false, nil
	}
	for pos, piece := range child.Board {
		if piece&0xF == pawn|1 && board.Board[pos]&0xF != pawn|1 {
			game.MovesSincePawn = 0
			return true, nil
		}
	}
	return true, nil
}

func (game *Game) putBoard(state chessState) error {
	if game.End {
		return echo.NewHTTPError(http.StatusBadRequest, "game is over")
	}
	if valid, err := game.validMove(state); !valid {
		if err != nil {
			return err
		}
		return echo.NewHTTPError(http.StatusBadRequest, "invalid move")
	}
	board, err := getBoardByBoard(state)
	if err != nil {
		return err
	}
	game.InactiveAgent, game.ActiveAgent = game.ActiveAgent, game.InactiveAgent
	game.InactiveAgentType, game.ActiveAgentType = game.ActiveAgentType, game.InactiveAgentType
	game.ActiveAgentPurple = !game.ActiveAgentPurple
	game.MoveCount = game.MoveCount + 1
	game.MovesSincePawn = game.MovesSincePawn + 1
	game.Board = board
	game.End = game.MoveCount > 4048 || game.MovesSincePawn > 50 || board.end()
	if game.End {
		if board.end() {
			board.ActiveScore = 3
			board.InactiveScore = -2
		} else {
			board.ActiveScore = 1
			board.InactiveScore = 1
		}
		if err := db.Save(&board).Error; err != nil {
			return err
		}
	} else {
		if err := board.lookahead3(game.ActiveAgentPurple); err != nil {
			return err
		}
	}
	if err := db.Save(&game).Error; err != nil {
		return err
	}
	if game.InactiveAgentType == game.ActiveAgentType {
		go func() {
			idleError("poke agent", game.pokeAgent())
		}()
	} else {
		return game.pokeAgent()
	}
	return nil
}

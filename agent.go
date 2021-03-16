package main

import (
	"crypto/rand"
	"math"
	"math/big"
	"net/http"
	"time"

	"github.com/apex/log"
	"github.com/labstack/echo/v4"
	"github.com/montanaflynn/stats"
	uuid "github.com/satori/go.uuid"
	"gorm.io/gorm/clause"
)

func agentIdle() error {
	var games []Game
	thirtySecondsAgo := time.Now().Add(time.Second * -30)
	if err := db.Where("updated_at < ?", thirtySecondsAgo).Not(Game{ActiveAgentType: "user"}).Not(db.Where(Game{InactiveAgent: placeHolder}).Or(Game{End: true})).Find(&games).Error; err != nil {
		return err
	}
	for _, game := range games {
		if err := game.pokeAgent(); err != nil {
			return err
		}
	}
	var count int64
	if err := db.Model(&Game{}).Not(db.Where(Game{ActiveAgentType: "user"}).Or(Game{InactiveAgentType: "user"})).Not(db.Where(Game{InactiveAgent: placeHolder}).Or(Game{End: true})).Count(&count).Error; err != nil {
		return err
	}
	if count < 5 {
		if err := db.Where(Game{InactiveAgent: placeHolder}).Find(&games).Error; err != nil {
			return err
		}
		newGames := 3
		if newGames >= len(games) {
			newGames = len(games)
		}
		for i := 0; i < newGames; i++ {
			if _, err := games[i].makeAgent("agent"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (game *Game) makeAgent(agentType string) (uuid.UUID, error) {
	id := uuid.NewV4()
	if err := game.addAgent(id, agentType); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func getAgent(id uuid.UUID) (*Game, error) {
	var game Game
	if err := db.Preload(clause.Associations).Where(Game{ActiveAgent: id}).Or(Game{InactiveAgent: id}).First(&game).Error; err != nil {
		return nil, err
	}
	return &game, nil
}

func (game *Game) playRound(id uuid.UUID, state *chessState) error {
	if !uuid.Equal(id, game.ActiveAgent) {
		return echo.NewHTTPError(http.StatusNotAcceptable, "not your turn")
	}
	if game.ActiveAgentType == "user" {
		if state == nil {
			return echo.NewHTTPError(http.StatusNotAcceptable, "player must provide move")
		}
		return game.putBoard(*state)
	}
	if game.End {
		return nil
	}
	board, err := getBoard(game.BoardID)
	if err != nil {
		return err
	}
	if len(board.Children) == 0 {
		return echo.NewHTTPError(http.StatusNotAcceptable, "no moves available")
	}
	return game.putBoard(decide(board.Children))
}

func decide(boards []Board) chessState {
	scores := make([]int, 0, len(boards))
	for _, board := range boards {
		scores = append(scores, board.InactiveScore)
	}
	percentile, err := stats.Percentile(stats.LoadRawData(scores), 80)
	if err != nil {
		log.WithError(err).Error("error")
		panic(err)
	}
	lowScore := int(math.Round(percentile))
	choices := make([]chessState, 0, len(boards)*len(boards))
	for _, board := range boards {
		if lowScore <= board.InactiveScore {
			count := (board.InactiveScore - lowScore) + 1
			if count > len(boards) {
				count = len(boards)
			}
			for i := 0; i < count; i++ {
				choices = append(choices, board.Board)
			}
		}
	}
	choice, err := rand.Int(rand.Reader, big.NewInt(int64(len(choices))))
	if err != nil {
		log.WithError(err).Error("error")
		panic(err)
	}
	return choices[choice.Uint64()]
}

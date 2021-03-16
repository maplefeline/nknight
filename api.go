package main

import (
	"errors"
	"net/http"
	"path"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	uuid "github.com/satori/go.uuid"
	"gorm.io/gorm"
)

type agentRequest struct {
	Type   string
	GameID uuid.UUID
}

type playRequest struct {
	Board *chessState
	Move  *move
}

type boardResponse struct {
	Href  string
	Board Board
}

type gameResponse struct {
	Href string
	Game Game
}

type gamesResponse struct {
	Href  string
	Games []Game
}

type playsResponse struct {
	Href   string
	Boards []chessState
	Moves  []move
}

func errToHTTP(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return echo.ErrNotFound
	}
	return err
}

func requestID(c echo.Context) (uuid.UUID, error) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return id, nil
}

func requestAgent(c echo.Context) (*Game, uuid.UUID, error) {
	id, err := requestID(c)
	if err != nil {
		return nil, uuid.Nil, err
	}
	game, err := getAgent(id)
	return game, id, err
}

func requestGame(c echo.Context) (*Game, error) {
	id, err := requestID(c)
	if err != nil {
		return nil, err
	}
	return getGame(id)
}

func responseAgent(game *Game, agentID uuid.UUID) gameResponse {
	return gameResponse{Game: game.response(agentID), Href: path.Join("/agents", agentID.String())}
}

func responseGame(game *Game) gameResponse {
	return gameResponse{Game: game.response(uuid.Nil), Href: path.Join("/games", game.GameID.String())}
}

func responseGames(games []Game) gamesResponse {
	for _, game := range games {
		game.response(uuid.Nil)
	}
	return gamesResponse{Games: games, Href: "/games"}
}

func responsePlays(game *Game, boards []chessState, moves []move) playsResponse {
	return playsResponse{Boards: boards, Moves: moves, Href: path.Join("/games", game.GameID.String(), "plays")}
}

func apiHandler() *echo.Echo {
	e := echo.New()

	e.POST("/agents", func(c echo.Context) error {
		var message agentRequest
		if err := c.Bind(&message); err != nil {
			return err
		}
		if message.Type == "" {
			message.Type = "agent"
		}
		game, err := getGame(message.GameID)
		if err != nil {
			return errToHTTP(err)
		}
		id, err := game.makeAgent(message.Type)
		if err != nil {
			return errToHTTP(err)
		}
		return c.JSON(http.StatusCreated, responseAgent(game, id))
	})
	e.GET("/agents/:id", func(c echo.Context) error {
		game, id, err := requestAgent(c)
		if err != nil {
			return errToHTTP(err)
		}
		return c.JSON(http.StatusOK, responseAgent(game, id))
	})
	e.PUT("/agents/:id", func(c echo.Context) error {
		game, id, err := requestAgent(c)
		if err != nil {
			return errToHTTP(err)
		}
		var request playRequest
		if err := c.Bind(&request); err != nil {
			return err
		}
		if request.Board == nil && request.Move != nil {
			request.Board = game.moveToBoard(*request.Move)
		}
		if err := game.playRound(id, request.Board); err != nil {
			return errToHTTP(err)
		}
		return c.JSON(http.StatusOK, responseAgent(game, id))
	})
	e.POST("/agents/:id", func(c echo.Context) error {
		game, id, err := requestAgent(c)
		if err != nil {
			return errToHTTP(err)
		}
		return c.JSON(http.StatusOK, responseAgent(game, id))
	})
	e.GET("/games", func(c echo.Context) error {
		games, err := getGames()
		if err != nil {
			return errToHTTP(err)
		}
		return c.JSON(http.StatusOK, responseGames(games))
	})
	e.POST("/games", func(c echo.Context) error {
		game, err := makeGame()
		if err != nil {
			return errToHTTP(err)
		}
		return c.JSON(http.StatusCreated, responseGame(game))
	})
	e.GET("/games/:id", func(c echo.Context) error {
		game, err := requestGame(c)
		if err != nil {
			return errToHTTP(err)
		}
		return c.JSON(http.StatusOK, responseGame(game))
	})
	e.GET("/games/:id/plays", func(c echo.Context) error {
		game, err := requestGame(c)
		if err != nil {
			return errToHTTP(err)
		}
		boards, moves, err := game.getPlays()
		if err != nil {
			return errToHTTP(err)
		}
		return c.JSON(http.StatusOK, responsePlays(game, boards, moves))
	})

	e.File("/", "static/index.html")
	e.File("/favicon.ico", "images/favicon.ico")
	e.Static("/static", "static")

	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Gzip())
	e.Use(middleware.RequestID())
	e.Use(middleware.Secure())
	e.Use(middleware.Static("/static"))

	return e
}

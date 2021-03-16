package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"
	. "gopkg.in/check.v1"
)

type greaterThanChecker struct {
	*CheckerInfo
}

func (checker *greaterThanChecker) Check(params []interface{}, names []string) (result bool, error string) {
	defer func() {
		if v := recover(); v != nil {
			result = false
			error = fmt.Sprint(v)
		}
	}()

	a, aOk := params[0].(int)
	b, bOk := params[1].(int)

	return aOk && bOk && a > b, ""
}

var greaterThan Checker = &greaterThanChecker{
	&CheckerInfo{Name: "greaterThan", Params: []string{"obtained", "expected"}},
}

func Test(t *testing.T) { TestingT(t) }

var jsonHeader = "application/json; charset=UTF-8"
var invalidUUID = "00600006+8600+4020+8711+600510061050"
var invalidUUIDErr string
var invalidAgent string
var invalidGame string
var invalidGamePlays string
var unknownUUID = "00600006-8600-4020-8711-600510061050"
var unknownAgent string
var unknownGame string
var unknownGamePlays string

func init() {
	invalidUUIDErr = strings.Join([]string{"uuid: incorrect UUID format", invalidUUID}, " ")
	invalidAgent = path.Join("agents", invalidUUID)
	invalidGame = path.Join("games", invalidUUID)
	invalidGamePlays = path.Join(invalidGame, "plays")
	unknownAgent = path.Join("agents", unknownUUID)
	unknownGame = path.Join("games", unknownUUID)
	unknownGamePlays = path.Join(unknownGame, "plays")
}

type echoErrorResponse struct {
	Message string
}

type NKnightSuite struct {
	srv      *httptest.Server
	client   *http.Client
	endpoint *url.URL
}

var _ = Suite(&NKnightSuite{})

func (s *NKnightSuite) SetUpSuite(c *C) {
	s.srv = httptest.NewServer(apiHandler())
	s.client = s.srv.Client()
	endpoint, err := url.Parse(s.srv.URL)
	c.Assert(err, IsNil)
	s.endpoint = endpoint
}

func (s *NKnightSuite) SetUpTest(c *C) {
}

func (s *NKnightSuite) TearDownTest(c *C) {
	c.Assert(db.Exec("DELETE FROM games").Error, IsNil)
}

func (s *NKnightSuite) TearDownSuite(c *C) {
	s.srv.Close()
	c.Assert(Close(), IsNil)
}

func (s NKnightSuite) makeURLString(c *C, input string) string {
	uriURL, err := url.Parse(input)
	c.Assert(err, IsNil)
	uriURL = s.endpoint.ResolveReference(uriURL)
	return uriURL.String()
}

func (s *NKnightSuite) doHTTP(c *C, method string, path string, request interface{}) *http.Response {
	buffer, err := json.Marshal(request)
	c.Assert(err, IsNil)
	req, err := http.NewRequest(method, s.makeURLString(c, path), bytes.NewReader(buffer))
	req.Header.Add("Content-Type", jsonHeader)
	c.Assert(err, IsNil)
	res, err := s.client.Do(req)
	c.Assert(err, IsNil)
	return res
}

func (s *NKnightSuite) delete(c *C, path string) *http.Response {
	return s.doHTTP(c, http.MethodDelete, path, nil)
}

func (s *NKnightSuite) get(c *C, path string) *http.Response {
	res, err := s.client.Get(s.makeURLString(c, path))
	c.Assert(err, IsNil)
	return res
}

func (s *NKnightSuite) post(c *C, path string, request interface{}) *http.Response {
	res, err := s.client.Post(s.makeURLString(c, path), jsonHeader, s.requestJSON(c, request))
	c.Assert(err, IsNil)
	return res
}

func (s *NKnightSuite) put(c *C, path string, request interface{}) *http.Response {
	return s.doHTTP(c, http.MethodPut, path, request)
}

func (s *NKnightSuite) requestJSON(c *C, request interface{}) io.Reader {
	buffer, err := json.Marshal(request)
	c.Assert(err, IsNil)
	return bytes.NewReader(buffer)
}

func (s *NKnightSuite) responseJSON(c *C, res *http.Response, response interface{}) {
	c.Assert(res.Header.Get("Content-Type"), Equals, jsonHeader)
	buffer, err := ioutil.ReadAll(res.Body)
	c.Assert(err, IsNil)
	err = json.Unmarshal(buffer, response)
	c.Assert(err, IsNil)
}

func (s *NKnightSuite) responseError(c *C, res *http.Response, code int, message string) {
	c.Assert(res.StatusCode, Equals, code)
	var response echoErrorResponse
	s.responseJSON(c, res, &response)
	c.Assert(response.Message, Equals, message)
}

func (s *NKnightSuite) response200(c *C, res *http.Response, response interface{}) {
	c.Assert(res.StatusCode, Equals, 200)
	s.responseJSON(c, res, response)
}

func (s *NKnightSuite) response400(c *C, res *http.Response, message string) {
	s.responseError(c, res, 400, message)
}

func (s *NKnightSuite) response404(c *C, res *http.Response) {
	s.responseError(c, res, 404, "Not Found")
}

func (s *NKnightSuite) response405(c *C, res *http.Response) {
	s.responseError(c, res, 405, "Method Not Allowed")
}

func (s *NKnightSuite) response406(c *C, res *http.Response, message string) {
	s.responseError(c, res, 406, message)
}

func (s *NKnightSuite) delete404(c *C, path string) {
	res := s.delete(c, path)
	defer res.Body.Close()
	s.response404(c, res)
}

func (s *NKnightSuite) delete405(c *C, path string) {
	res := s.delete(c, path)
	defer res.Body.Close()
	s.response405(c, res)
}

func (s *NKnightSuite) get200(c *C, path string, response interface{}) {
	res := s.get(c, path)
	defer res.Body.Close()
	s.response200(c, res, response)
}

func (s *NKnightSuite) get400(c *C, path string, message string) {
	res := s.get(c, path)
	defer res.Body.Close()
	s.response400(c, res, message)
}

func (s *NKnightSuite) get404(c *C, path string) {
	res := s.get(c, path)
	defer res.Body.Close()
	s.response404(c, res)
}

func (s *NKnightSuite) get405(c *C, path string) {
	res := s.get(c, path)
	defer res.Body.Close()
	s.response405(c, res)
}

func (s *NKnightSuite) post201(c *C, path string, request interface{}, response interface{}) {
	res := s.post(c, path, request)
	defer res.Body.Close()
	c.Assert(res.StatusCode, Equals, 201)
	s.responseJSON(c, res, response)
}

func (s *NKnightSuite) post400(c *C, path string, request interface{}, message string) {
	res := s.post(c, path, request)
	defer res.Body.Close()
	s.response400(c, res, message)
}

func (s *NKnightSuite) post404(c *C, path string, request interface{}) {
	res := s.post(c, path, request)
	defer res.Body.Close()
	s.response404(c, res)
}

func (s *NKnightSuite) post405(c *C, path string, request interface{}) {
	res := s.post(c, path, request)
	defer res.Body.Close()
	s.response405(c, res)
}

func (s *NKnightSuite) put200(c *C, path string, request interface{}, response interface{}) {
	res := s.put(c, path, request)
	defer res.Body.Close()
	s.response200(c, res, response)
}

func (s *NKnightSuite) put400(c *C, path string, request interface{}, message string) {
	res := s.put(c, path, request)
	defer res.Body.Close()
	s.response400(c, res, message)
}

func (s *NKnightSuite) put404(c *C, path string, request interface{}) {
	res := s.put(c, path, request)
	defer res.Body.Close()
	s.response404(c, res)
}

func (s *NKnightSuite) put405(c *C, path string) {
	res := s.put(c, path, nil)
	defer res.Body.Close()
	s.response405(c, res)
}

func (s *NKnightSuite) put406(c *C, path string, request interface{}, message string) {
	res := s.put(c, path, request)
	defer res.Body.Close()
	s.response406(c, res, message)
}

func (s *NKnightSuite) generateGame(c *C) *gameResponse {
	var response gameResponse
	s.post201(c, "games", nil, &response)
	c.Assert(response.Game, NotNil)
	return &response
}

func (s *NKnightSuite) addUser(c *C, id uuid.UUID) *gameResponse {
	var response gameResponse
	s.post201(c, "agents", agentRequest{Type: "user", GameID: id}, &response)
	c.Assert(response.Game, NotNil)
	return &response
}

func (s *NKnightSuite) addAgent(c *C, id uuid.UUID) *gameResponse {
	var response gameResponse
	s.post201(c, "agents", agentRequest{Type: "agent", GameID: id}, &response)
	c.Assert(response.Game, NotNil)
	return &response
}

func (s *NKnightSuite) TestFmtBoard(c *C) {
	value, err := chessState{}.Value()
	c.Assert(err, IsNil)
	c.Assert(value, Equals, "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")
	value, err = initialBoard.Value()
	c.Assert(err, IsNil)
	c.Assert(value, Equals, "0d07030b0503070d0909090909090909000000000000000000000000000000000000000000000000000000000000000008080808080808080c06020a0402060c")
}

// func (s *NKnightSuite) TestGetGames(c *C) {
// 	var response gamesResponse
// 	s.get200(c, "games", &response)
// 	c.Assert(response.Games, HasLen, 0)
// }
//
// func (s *NKnightSuite) TestPostGames(c *C) {
// 	game := s.generateGame(c)
// 	var response gameResponse
// 	s.get200(c, game.Href, &response)
// 	c.Assert(response.Href, Equals, game.Href)
// }
//
// func (s *NKnightSuite) TestGetPlays(c *C) {
// 	game := s.generateGame(c)
// 	href := path.Join(game.Href, "plays")
// 	var response playsResponse
// 	s.get200(c, href, &response)
// 	c.Assert(response.Href, Equals, href)
// 	c.Assert(len(response.Boards), Equals, 20)
// }
//
// func (s *NKnightSuite) TestPostAgents(c *C) {
// 	game := s.generateGame(c)
// 	agent1 := s.addAgent(c, game.Game.GameID)
// 	var response gameResponse
// 	s.get200(c, agent1.Href, &response)
// 	c.Assert(response.Href, Equals, agent1.Href)
// 	// agent2 := s.addUser(c, game.Game.GameID)
// 	// s.get200(c, agent2.Href, &response)
// 	// c.Assert(response.Href, Equals, agent2.Href)
// }

// func (s *NKnightSuite) TestPlayBadGame(c *C) {
// 	game := s.generateGame(c)
// 	agent1 := s.addUser(c, game.Game.GameID)
// 	agent2 := s.addUser(c, game.Game.GameID)
// 	rawRequest := map[string]interface{}{"Board": 0}
// 	s.put400(c, agent1.Href, &rawRequest, "Unmarshal type error: expected=main.chessState, got=number, field=Board, offset=10")
// 	s.put400(c, agent2.Href, &rawRequest, "Unmarshal type error: expected=main.chessState, got=number, field=Board, offset=10")
// 	rawRequest = map[string]interface{}{"Board": "Hello World!"}
// 	s.put400(c, agent1.Href, &rawRequest, "board is not length 8: 1")
// 	s.put400(c, agent2.Href, &rawRequest, "board is not length 8: 1")
// 	rawRequest = map[string]interface{}{"Board": ",,,,,,,"}
// 	s.put400(c, agent1.Href, &rawRequest, "row 0 is not length 8: 0")
// 	s.put400(c, agent2.Href, &rawRequest, "row 0 is not length 8: 0")
// 	rawRequest = map[string]interface{}{"Board": "  ,,,,,,,"}
// 	s.put400(c, agent1.Href, &rawRequest, "encoding/hex: invalid byte: U+0020 ' '")
// 	s.put400(c, agent2.Href, &rawRequest, "encoding/hex: invalid byte: U+0020 ' '")
// 	request := playRequest{Board: &initialBoard}
// 	s.put400(c, agent1.Href, &request, "invalid move")
// 	s.put406(c, agent2.Href, &request, "not your turn")
// 	request = playRequest{Board: &chessState{}}
// 	s.put400(c, agent1.Href, &request, "invalid move")
// 	s.put406(c, agent2.Href, &request, "not your turn")
// 	s.put406(c, agent1.Href, &playRequest{}, "player must provide move")
// 	s.put406(c, agent2.Href, &playRequest{}, "not your turn")
// }

// func (s *NKnightSuite) TestPlayAIBadGame(c *C) {
// 	game := s.generateGame(c)
// 	s.addAgent(c, game.Game.GameID)
// 	agent2 := s.addUser(c, game.Game.GameID)
// 	s.put406(c, agent2.Href, &playRequest{}, "player must provide move")
// 	s.put400(c, agent2.Href, &playRequest{Board: &initialBoard}, "invalid move")
// }

// func (s *NKnightSuite) TestPlayFoolsMateGame(c *C) {
// 	game := s.generateGame(c)
// 	href := path.Join(game.Href, "plays")
// 	agent1 := s.addUser(c, game.Game.GameID)
// 	agent2 := s.addUser(c, game.Game.GameID)
// 	var state gameResponse
// 	var response playsResponse
// 	s.get200(c, href, &response)
// 	c.Assert(response.Moves, HasLen, 20)
// 	s.put200(c, agent1.Href, &map[string]string{"Move": "♟f2f3"}, &state)
// 	s.get200(c, href, &response)
// 	c.Assert(response.Moves, HasLen, 20)
// 	s.put200(c, agent2.Href, &map[string]string{"Move": "♟e2e3"}, &state)
// 	s.get200(c, href, &response)
// 	c.Assert(response.Moves, HasLen, 19)
// 	s.put200(c, agent1.Href, &map[string]string{"Move": "♟g2g4"}, &state)
// 	s.get200(c, href, &response)
// 	c.Assert(response.Moves, HasLen, 30)
// 	s.put200(c, agent2.Href, &map[string]string{"Move": "♛d8h5"}, &state)
// 	s.get200(c, href, &response)
// 	c.Assert(response.Moves, HasLen, 0)
// }

// func (s *NKnightSuite) TestPlayOneRoundGame(c *C) {
// 	game := s.generateGame(c)
// 	href := path.Join(game.Href, "plays")
// 	var response playsResponse
// 	s.get200(c, href, &response)
// 	c.Assert(response.Boards, HasLen, 20)
// 	c.Assert(response.Moves, HasLen, 20)
// 	for _, play := range response.Boards {
// 		game := s.generateGame(c)
// 		agent1 := s.addUser(c, game.Game.GameID)
// 		s.addAgent(c, game.Game.GameID)
// 		var state gameResponse
// 		s.put200(c, agent1.Href, &playRequest{Board: &play}, &state)
// 		c.Assert(state.Href, Equals, agent1.Href)
// 		c.Assert(state.Game.MoveCount, Equals, 2)
// 		c.Assert(state.Game.ActiveAgent, Not(DeepEquals), uuid.Nil)
// 		c.Assert(state.Game.InactiveAgent, DeepEquals, uuid.Nil)
// 	}
// 	for _, play := range response.Moves {
// 		game := s.generateGame(c)
// 		agent1 := s.addUser(c, game.Game.GameID)
// 		s.addAgent(c, game.Game.GameID)
// 		var state gameResponse
// 		s.put200(c, agent1.Href, &playRequest{Move: &play}, &state)
// 		c.Assert(state.Href, Equals, agent1.Href)
// 		c.Assert(state.Game.MoveCount, Equals, 2)
// 		c.Assert(state.Game.ActiveAgent, Not(DeepEquals), uuid.Nil)
// 		c.Assert(state.Game.InactiveAgent, DeepEquals, uuid.Nil)
// 	}
// }

// func (s *NKnightSuite) TestPlayFullGame(c *C) {
// 	game := s.generateGame(c)
// 	s.addAgent(c, game.Game.GameID)
// 	s.addAgent(c, game.Game.GameID)
// 	var response gameResponse
// 	for {
// 		s.get200(c, game.Href, &response)
// 		if response.Game.End {
// 			break
// 		}
// 		c.Assert(response.Game.ActiveAgent, DeepEquals, uuid.Nil)
// 		c.Assert(response.Game.InactiveAgent, DeepEquals, uuid.Nil)
// 		time.Sleep(5 * time.Second)
// 	}
// 	c.Assert(response.Game.ActiveAgent, Not(DeepEquals), uuid.Nil)
// 	c.Assert(response.Game.InactiveAgent, Not(DeepEquals), uuid.Nil)
// 	c.Assert(response.Game.MoveCount, greaterThan, 5)
// 	c.Assert(response.Game.MovesSincePawn, greaterThan, 0)
// }

func (s *NKnightSuite) TestDeleteBadURL(c *C) {
	s.delete404(c, "foo")
}

func (s *NKnightSuite) TestGetBadURL(c *C) {
	s.get404(c, "foo")
}

func (s *NKnightSuite) TestPostBadURL(c *C) {
	s.post404(c, "foo", nil)
}

func (s *NKnightSuite) TestPutBadURL(c *C) {
	s.put404(c, "foo", nil)
}

func (s *NKnightSuite) TestDeleteIndex(c *C) {
	s.delete405(c, "")
}

func (s *NKnightSuite) TestPostIndex(c *C) {
	s.post405(c, "", nil)
}

func (s *NKnightSuite) TestPutIndex(c *C) {
	s.put405(c, "")
}

func (s *NKnightSuite) TestPutGames(c *C) {
	s.put405(c, "games")
}

func (s *NKnightSuite) TestDeleteGames(c *C) {
	s.delete405(c, "games")
}

func (s *NKnightSuite) TestDeleteGameUnknownID(c *C) {
	s.delete405(c, unknownGame)
}

func (s *NKnightSuite) TestGetGameInvaidID(c *C) {
	s.get400(c, invalidGame, invalidUUIDErr)
}

func (s *NKnightSuite) TestGetGameUnknownID(c *C) {
	s.get404(c, unknownGame)
}

func (s *NKnightSuite) TestPostGameUnknownID(c *C) {
	s.post405(c, unknownGame, nil)
}

func (s *NKnightSuite) TestPutGameUnknownID(c *C) {
	s.put405(c, unknownGame)
}

func (s *NKnightSuite) TestDeleteGamePlaysUnknownID(c *C) {
	s.delete405(c, unknownGamePlays)
}

func (s *NKnightSuite) TestGetGamePlaysInvaidID(c *C) {
	s.get400(c, invalidGamePlays, invalidUUIDErr)
}

func (s *NKnightSuite) TestGetGamePlaysUnknownID(c *C) {
	s.get404(c, unknownGamePlays)
}

func (s *NKnightSuite) TestPostGamePlaysUnknownID(c *C) {
	s.post405(c, unknownGamePlays, nil)
}

func (s *NKnightSuite) TestPutGamePlaysUnknownID(c *C) {
	s.put405(c, unknownGamePlays)
}

func (s *NKnightSuite) TestGetAgents(c *C) {
	s.get405(c, "agents")
}

func (s *NKnightSuite) TestPostAgentsInvalidUUIDGame(c *C) {
	s.post400(c, "agents", map[string]interface{}{"GameID": invalidUUID}, invalidUUIDErr)
}

func (s *NKnightSuite) TestPostAgentsUnknownIDGame(c *C) {
	id, err := uuid.FromString(unknownUUID)
	c.Assert(err, IsNil)
	s.post404(c, "agents", agentRequest{GameID: id})
}

func (s *NKnightSuite) TestPutAgents(c *C) {
	s.put405(c, "agents")
}

func (s *NKnightSuite) TestDeleteAgents(c *C) {
	s.delete405(c, "agents")
}

func (s *NKnightSuite) TestDeleteAgentUnknownID(c *C) {
	s.delete405(c, unknownAgent)
}

func (s *NKnightSuite) TestGetAgentInvalidUUID(c *C) {
	s.get400(c, invalidAgent, invalidUUIDErr)
}

func (s *NKnightSuite) TestGetAgentUnknownID(c *C) {
	s.get404(c, unknownAgent)
}

func (s *NKnightSuite) TestPostAgentInvalidUUID(c *C) {
	s.post400(c, invalidAgent, nil, invalidUUIDErr)
}

func (s *NKnightSuite) TestPostAgentUnknownID(c *C) {
	s.post404(c, unknownAgent, nil)
}

func (s *NKnightSuite) TestPutAgentInvalidUUID(c *C) {
	s.put400(c, invalidAgent, nil, invalidUUIDErr)
}

func (s *NKnightSuite) TestPutAgentUnknownID(c *C) {
	s.put404(c, unknownAgent, nil)
}

// func (s *NKnightSuite) TestIdle(c *C) {
// 	c.Assert(gameIdle(), IsNil)
// 	c.Assert(agentIdle(), IsNil)
// }

func (s *NKnightSuite) TestShutdown(c *C) {
	closed := make(chan interface{})
	go waitShutdown(apiHandler(), closed)
	select {
	case res := <-closed:
		c.Assert(res, IsNil)
		c.Fail()
	case <-time.After(1 * time.Second):
		close(sigint)
		<-closed
	}
}

func (s *NKnightSuite) TestListenAndServe(c *C) {
	closed := make(chan interface{})
	go listenAndServe(":3000", closed)
	select {
	case res := <-closed:
		c.Assert(res, IsNil)
		c.Fail()
	case <-time.After(1 * time.Second):
		close(sigint)
		<-closed
	}
}

func (s *NKnightSuite) TestOpen(c *C) {
	go Open(":3001")
	<-time.After(1 * time.Second)
	close(sigint)
	<-time.After(1 * time.Second)
}

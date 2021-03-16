import cmd
import json
import pathlib
import urllib.parse
from typing import List, Optional

import requests

TRANSLATION = {
    0: " ",
    2: "♝",
    3: "♗",
    4: "♚",
    5: "♔",
    6: "♞",
    7: "♘",
    8: "♟",
    9: "♙",
    10: "♛",
    11: "♕",
    12: "♜",
    13: "♖",
}
DARK_TRANSLATION = {id ^ 1: piece for id, piece in TRANSLATION.items()}
DARK_TRANSLATION[0] = " "
del DARK_TRANSLATION[1]

LIGHT_TO_DARK = str.maketrans("12345678", "87654321")
LIGHT_TO_DARK_SEND = LIGHT_TO_DARK
DARK_TO_LIGHT = str.maketrans("♚♛♜♝♞♟", "♔♕♖♗♘♙")
DARK_TO_LIGHT_SEND = str.maketrans("♔♕♖♗♘♙", "♚♛♜♝♞♟")


def dumps(obj):
    return json.dumps(obj, sort_keys=True, indent=4)


def print_r(r, debug=True):
    try:
        r.raise_for_status()
        data = r.json()
        if debug:
            print(dumps(data))
        return data
    except Exception:
        print(r.text)
        raise


class ChessShell(cmd.Cmd):
    agents: List[str] = []
    active: Optional[str] = None
    debug = False

    def format_row(self, uint, row):
        print(uint, " ".join("|".join(row)), "|")

    def format_board(self, is_purple, board):
        board = board.split(",")
        if is_purple:
            self.format_row(" ", "abcdefgh")
            ranks = range(8, -2, -1)
            translation = TRANSLATION
        else:
            self.format_row(" ", reversed("abcdefgh"))
            ranks = range(1, 10)
            translation = DARK_TRANSLATION
        for uint, row in zip(ranks, board):
            row = (translation[piece & 0xF] for piece in bytearray.fromhex(row))
            self.format_row(uint, row)

    def _urljoin(self, arg):
        return urllib.parse.urljoin("http://localhost:8080", arg)

    def do_get(self, arg, debug=None, **kwargs):
        if debug is None:
            debug = self.debug
        return print_r(requests.get(self._urljoin(arg)), debug=debug)

    def do_post(self, arg, debug=None, **kwargs):
        if debug is None:
            debug = self.debug
        return print_r(requests.post(self._urljoin(arg), json=kwargs), debug=debug)

    def do_put(self, arg, debug=None, **kwargs):
        if debug is None:
            debug = self.debug
        return print_r(requests.put(self._urljoin(arg), json=kwargs), debug=debug)

    def do_debug(self, arg):
        self.debug = bool(arg)

    def do_new(self, arg):
        type, *arg = arg.split()
        if type == "agent":
            if arg:
                (id,) = arg
            else:
                data = self.do_get("games")["Games"]
                if data:
                    id = data[0]["GameID"]
                else:
                    id = self.do_post("games")["Game"]["GameID"]
                    self.do_post("agents", GameID=id)
            self.agents.append(self.do_post("agents", Type="user", GameID=id)["Href"])
        elif type == "game":
            self.do_post("games", debug=True)

    def do_show(self, arg):
        type, *arg = arg.split()
        if type == "agents":
            print(dumps(self.agents))
        elif type == "board":
            game = self.do_get(self.active)["Game"]
            agent_id = self.active.split("/")[-1]
            active_agent = agent_id == game["ActiveAgent"]
            is_purple = active_agent == game["ActiveAgentPurple"]
            self.format_board(is_purple, game["Board"]["Board"])
        elif type == "games":
            self.do_get("games", debug=True)
        elif type == "plays":
            game = self.do_get(self.active)["Game"]
            agent_id = self.active.split("/")[-1]
            active_agent = agent_id == game["ActiveAgent"]
            is_purple = active_agent == game["ActiveAgentPurple"]
            game_id = game["GameID"]
            moves = self.do_get(f"games/{game_id}/plays")["Moves"]
            if is_purple:
                moves = map(lambda move: move.translate(DARK_TO_LIGHT), moves)
            else:
                moves = map(lambda move: move.translate(LIGHT_TO_DARK), moves)
            print(*moves, sep="\n")

    def do_play(self, arg):
        (move,) = arg.split()
        game = self.do_get(self.active)["Game"]
        agent_id = self.active.split("/")[-1]
        active_agent = agent_id == game["ActiveAgent"]
        is_purple = active_agent == game["ActiveAgentPurple"]
        if is_purple:
            move = move.translate(DARK_TO_LIGHT_SEND)
        else:
            move = move.translate(LIGHT_TO_DARK_SEND)
        self.do_put(self.active, Move=move)

    def do_activate(self, arg):
        self.active = self.agents[int(arg)]

    def precmd(self, line):
        return line.lower().strip()


if __name__ == "__main__":
    shell = ChessShell()
    try:
        shell.agents = json.loads(pathlib.Path("agents.json").read_text())
    except json.decoder.JSONDecodeError:
        pass
    except FileNotFoundError:
        pass
    try:
        shell.cmdloop()
    except KeyboardInterrupt:
        print()
    finally:
        pathlib.Path("agents.json").write_text(dumps(shell.agents))

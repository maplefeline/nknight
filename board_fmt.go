package main

import (
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func rankAndFile(pos int) (byte, uint) {
	return byte('a' + pos/8), uint(1 + pos%8)
}

func position(file byte, rank uint) int {
	return int('a'-file)*8 + (1 - int(rank))
}

func (m *move) Scan(state fmt.ScanState, verb rune) error {
	token, err := state.Token(true, nil)
	if err != nil {
		return err
	}
	s := fmt.Sprintf("%s", token)
	if s == "0-0-0" {
		m.castling = 'q'
		return nil
	} else if s == "0-0" {
		m.castling = 'k'
		return nil
	}
	if len(s) >= 7 {
		if _, err := fmt.Sscanf(s[:3], "%c", &m.piece); err != nil {
			return fmt.Errorf("invalid move format %d %s: %w", len(s), s, err)
		}
		m.depart = position(s[3], uint(s[4]-'1'+1))
		if s[5] == 'x' {
			m.capture = true
			m.dest = position(s[6], uint(s[7]-'1'+1))
			if len(s) == 8 {
				return nil
			}
			if _, err := fmt.Sscanf(s[7:], "%c", &m.promotion); err != nil {
				return fmt.Errorf("invalid move format %d %s: %w", len(s), s, err)
			}
			return nil
		}
		m.dest = position(s[5], uint(s[5]-'1'+1))
		if len(s) == 7 {
			return nil
		}
		if _, err := fmt.Sscanf(s[6:], "%c", &m.promotion); err != nil {
			return fmt.Errorf("invalid move format %d %s: %w", len(s), s, err)
		}
		return nil
	}
	return fmt.Errorf("invalid move format %d %s", len(s), s)
}

func (m move) String() string {
	if m.castling == 'q' {
		return "0-0-0"
	} else if m.castling == 'k' {
		return "0-0"
	}
	capture := ""
	if m.capture {
		capture = "x"
	}
	promotion := ""
	if m.promotion != rune(0) {
		promotion = fmt.Sprintf("%c", m.promotion)
	}
	departFile, departRank := rankAndFile(m.depart)
	destFile, destRank := rankAndFile(m.dest)
	return fmt.Sprintf("%c%c%d%s%c%d%s", m.piece, departFile, departRank, capture, destFile, destRank, promotion)
}

func (m *move) UnmarshalJSON(bytes []byte) error {
	var state string
	if err := json.Unmarshal(bytes, &state); err != nil {
		return err
	}
	_, err := fmt.Sscan(state, m)
	return err
}

func (m move) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

func (board chessState) Value() (driver.Value, error) {
	return hex.EncodeToString(board[:]), nil
}

func (board *chessState) Scan(cell interface{}) error {
	switch cell := cell.(type) {
	case string:
		src, err := hex.DecodeString(cell)
		if err != nil {
			return err
		}
		copy(board[:], src)
	default:
		return fmt.Errorf("invalid format scaning %#v", cell)
	}
	return nil
}

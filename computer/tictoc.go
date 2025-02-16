package computer

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"unicode/utf8"
)

func getSlot(board uint32, number int) uint32 {
	return uint32(board>>(number<<1)) & 0b11
}

func putSlot(board uint32, number int, piece uint32) uint32 {
	return (board & ^(0b11 << (number << 1))) | uint32(piece<<(number<<1))
}

func WhichTurn(board uint32) int {
	t := 0
	for i := range 9 {
		if getSlot(board, i) != 0 {
			t++
		}
	}

	return t
}

// Board is as follows,
// 8 7 6
// 5 4 3
// 2 1 0
func hasWinner(board uint32) bool {
	stensils := [][]int{
		{8, 7, 6},
		{5, 4, 3},
		{2, 1, 0},

		{8, 5, 2},
		{7, 4, 1},
		{6, 3, 0},

		{8, 4, 0},
		{6, 4, 2},
	}

	for _, stensil := range stensils {
		winn := getSlot(board, stensil[0]) != 0 &&
			getSlot(board, stensil[0]) == getSlot(board, stensil[1]) &&
			getSlot(board, stensil[1]) == getSlot(board, stensil[2])

		if winn {
			return true
		}
	}

	return false
}

type Stats struct{ X, O, T uint32 }

func (s Stats) add(nextStats Stats) Stats {
	return Stats{
		X: s.X + nextStats.X,
		O: s.O + nextStats.O,
		T: s.T + nextStats.T,
	}
}

type Computer struct {
	tree     map[uint32][]uint32
	statsMap map[uint32]Stats
}

func (c Computer) Tree() map[uint32][]uint32 {
	return maps.Clone(c.tree)
}

func (c Computer) Next(board uint32) []uint32 {
	return c.tree[board]
}

func (c Computer) Stats(board uint32) Stats {
	return c.statsMap[board]
}

func (c Computer) StatsMap() map[uint32]Stats {
	return maps.Clone(c.statsMap)
}

func (c Computer) walkRecurse(current uint32, level uint32) Stats {
	if level > 4 && hasWinner(current) {
		if level%2 == 0 {
			return Stats{X: 0, O: 1, T: 0}
		}

		return Stats{X: 1, O: 0, T: 0}
	} else if level == 9 {
		return Stats{X: 0, O: 0, T: 1}
	}

	var stats Stats
	for i := range 9 {
		if getSlot(current, i) != 0 {
			continue
		}

		next := putSlot(current, i, level%2+1)
		c.tree[current] = append(c.tree[current], next)

		nextStats := c.walkRecurse(next, level+1)
		c.statsMap[next] = nextStats
		stats = stats.add(nextStats)
	}

	return stats
}

func Walk() Computer {
	c := Computer{
		tree:     map[uint32][]uint32{},
		statsMap: map[uint32]Stats{},
	}

	stats := c.walkRecurse(0, 0)
	c.statsMap[0] = stats

	return c
}

const magic uint64 = 0xb5dead0115145103

func (c Computer) Serialize(w io.Writer) error {
	if err := binary.Write(w, binary.BigEndian, int32(len(c.statsMap))); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, int32(len(c.tree))); err != nil {
		return err
	}

	if err := binary.Write(w, binary.BigEndian, magic); err != nil {
		return err
	}

	statsKeys := slices.Collect(maps.Keys(c.statsMap))
	slices.Sort(statsKeys)
	for _, k := range statsKeys {
		v := c.statsMap[k]
		if err := binary.Write(w, binary.BigEndian, k); err != nil {
			return err
		}

		if err := binary.Write(w, binary.BigEndian, v); err != nil {
			return err
		}
	}

	treeKeys := slices.Collect(maps.Keys(c.tree))
	slices.Sort(treeKeys)
	for _, node := range treeKeys {
		children := c.tree[node]

		if err := binary.Write(w, binary.BigEndian, node); err != nil {
			return err
		}

		if err := binary.Write(w, binary.BigEndian, int32(len(children))); err != nil {
			return err
		}

		if err := binary.Write(w, binary.BigEndian, children); err != nil {
			return err
		}
	}

	return nil
}

func Deserialize(r io.Reader) (Computer, error) {
	var (
		newTree     = make(map[uint32][]uint32)
		newStatsMap = make(map[uint32]Stats)
	)

	var statsLen, treeLen int32
	if err := binary.Read(r, binary.BigEndian, &statsLen); err != nil {
		return Computer{}, err
	}

	if err := binary.Read(r, binary.BigEndian, &treeLen); err != nil {
		return Computer{}, err
	}

	var magicValue uint64
	if err := binary.Read(r, binary.BigEndian, &magicValue); err != nil {
		return Computer{}, err
	}

	if magicValue != magic {
		return Computer{}, errors.New("corrupt file")
	}

	for range statsLen {
		var (
			k uint32
			v Stats
		)

		if err := binary.Read(r, binary.BigEndian, &k); err != nil {
			return Computer{}, err
		}

		if err := binary.Read(r, binary.BigEndian, &v); err != nil {
			return Computer{}, err
		}

		newStatsMap[k] = v
	}

	for range treeLen {
		var (
			node        uint32
			childrenLen int32
		)

		if err := binary.Read(r, binary.BigEndian, &node); err != nil {
			return Computer{}, err
		}

		if err := binary.Read(r, binary.BigEndian, &childrenLen); err != nil {
			return Computer{}, err
		}

		children := make([]uint32, childrenLen)
		for i := range children {
			err := binary.Read(r, binary.BigEndian, &children[i])
			if err != nil {
				return Computer{}, err
			}
		}

		newTree[node] = children
	}

	return Computer{tree: newTree, statsMap: newStatsMap}, nil
}

var (
	ErrLength      = errors.New("length must be 9")
	ErrIllegalChar = errors.New("illegal character")
)

func VectorToBoard(s string, seq [3]rune) (uint32, error) {
	if utf8.RuneCountInString(s) != 9 {
		return 0, ErrLength
	}

	i := 0 // Iterating through string with unicode jumps indices, hence the separate counter.
	var board uint32
	for _, c := range s {
		switch c {
		case seq[0]:
			// Do nothing
		case seq[1]:
			board = putSlot(board, i, 1)
		case seq[2]:
			board = putSlot(board, i, 2)
		default:
			return 0, fmt.Errorf("at position %v: %w", i, ErrIllegalChar)
		}
		i++
	}

	return board, nil
}

func BoardToVector(board uint32, seq [3]rune) string {
	s := make([]rune, 9) // Unicode rune can be up to 4 bytes
	for i := range 9 {
		c := getSlot(board, i)
		s[i] = seq[c]
	}

	return string(s)
}

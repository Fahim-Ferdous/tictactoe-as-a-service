package computer

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
)

func Test_getSlot(t *testing.T) {
	tests := []struct {
		name   string
		board  uint32
		number int
		want   uint32
	}{
		{
			name:   "empty",
			board:  0,
			number: 3,
			want:   0,
		},
		{
			name:   "first circ",
			board:  0b00000000000000000000000000010000,
			number: 2,
			want:   0b01,
		},
		{
			name:   "first cross",
			board:  0b00000000000000000000000000110000,
			number: 2,
			want:   0b11,
		},
		{
			name:   "third circ",
			board:  0b00000000000000000000000000010000,
			number: 2,
			want:   0b01,
		},
		{
			name:   "third cross",
			board:  0b00000000000000000000000000110000,
			number: 2,
			want:   0b11,
		},
		{
			name:   "ninth circ",
			board:  0b00000000000001000000000000010000,
			number: 9,
			want:   0b01,
		},
		{
			name:   "ninth cross",
			board:  0b00000000000011000000000000110000,
			number: 2,
			want:   0b11,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSlot(tt.board, tt.number)
			if got != tt.want {
				t.Errorf("getSlot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_putSlot(t *testing.T) {
	tests := []struct {
		name   string
		board  uint32
		number int
		piece  uint32
		want   uint32
	}{
		{
			name:   "empty first",
			board:  0,
			number: 0,
			piece:  2,
			want:   0b00000000000000000000000000000010,
		},
		{
			name:   "clear first",
			board:  0b00000000000000000000000000000010,
			number: 0,
			piece:  0,
			want:   0b00000000000000000000000000000000,
		},
		{
			name:   "empty ninth",
			board:  0b00000000000001000000000000000010,
			number: 9,
			piece:  0b10,
			want:   0b00000000000010000000000000000010,
		},
		{
			name:   "clear ninth",
			board:  0b00000000000011000000000000000010,
			number: 9,
			piece:  0,
			want:   0b00000000000000000000000000000010,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := putSlot(tt.board, tt.number, tt.piece)
			if got != tt.want {
				t.Errorf("mismatch\nputSlot() = %032b,\nwant\t%032b", got, tt.want)
			}
		})
	}
}

func Test_hasWinner(t *testing.T) {
	tests := []struct {
		name  string
		board uint32
		want  bool
	}{
		{
			name: "zero",
			board: 0 |
				0b_00_00_00<<12 |
				0b_00_00_00<<6 |
				0b_00_00_00<<0,
			want: false,
		},
		{
			name: "col",
			board: 0 |
				0b_01_00_00<<12 |
				0b_01_00_00<<6 |
				0b_01_00_00<<0,
			want: true,
		},
		{
			name: "diag 1",
			board: 0 |
				0b_01_00_00<<12 |
				0b_00_01_00<<6 |
				0b_00_00_01<<0,
			want: true,
		},
		{
			name: "diag 2",
			board: 0 |
				0b_00_00_01<<12 |
				0b_00_01_00<<6 |
				0b_01_00_00<<0,
			want: true,
		},
		{
			name: "row",
			board: 0 |
				0b_00_00_00<<12 |
				0b_00_00_00<<6 |
				0b_10_10_10<<0,
			want: true,
		},
		{
			name: "not",
			board: 0 |
				0b_00_00_00<<12 |
				0b_00_00_00<<6 |
				0b_01_00_01<<0,
			want: false,
		},
		{
			name: "mid not",
			board: 0 |
				0b_00_00_00<<12 |
				0b_00_11_00<<6 |
				0b_00_00_00<<0,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasWinner(tt.board)
			if got != tt.want {
				t.Errorf("hasWinner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputer_Serialize(t *testing.T) {
	tests := []struct {
		name  string
		c     Computer
		bytes []byte
	}{
		{
			name: "ez",
			c: Computer{
				tree: map[uint32][]uint32{
					123: {50, 50},
				},
				statsMap: map[uint32]Stats{
					123: {X: 1, O: 9, T: 8},
					69:  {X: 1, O: 6, T: 9},
				},
			},
			bytes: []byte{
				0, 0, 0, 2, // Stats length
				0, 0, 0, 1, // Tree length
				0xb5, 0xde, 0xad, 0x01, 0x15, 0x14, 0x51, 0x03, // Magic

				0, 0, 0, 69, // Key
				0, 0, 0, 1, // X
				0, 0, 0, 6, // O
				0, 0, 0, 9, // T
				0, 0, 0, 123, // Key
				0, 0, 0, 1, // X
				0, 0, 0, 9, // O
				0, 0, 0, 8, // T

				0, 0, 0, 123, // Key
				0, 0, 0, 2, // Out degree
				0, 0, 0, 50, // Child
				0, 0, 0, 50, // Child
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer([]byte{})
			err := tt.c.Serialize(buf)
			if err != nil {
				t.Errorf("Unexpected err: %v", err)
			}

			if !reflect.DeepEqual(buf.Bytes(), tt.bytes) {
				t.Logf("Output didnt match.\nExpected:\n%s\nGot:\n%s", hex.Dump(tt.bytes), hex.Dump(buf.Bytes()))
				t.Fail()
			}
		})
	}
}

func TestDeserialize(t *testing.T) {
	tests := []struct {
		name  string
		bytes []byte
		want  Computer
	}{
		{
			name: "ez",
			bytes: []byte{
				0, 0, 0, 2,
				0, 0, 0, 2,
				0xb5, 0xde, 0xad, 0x01, 0x15, 0x14, 0x51, 0x03, // Magic

				0, 0, 0, 42,
				0, 0, 0, 9,
				0, 0, 0, 8,
				0, 0, 0, 4,
				0, 0, 0, 10,
				0, 0, 0, 2,
				0, 0, 0, 4,
				0, 0, 0, 5,

				0, 0, 0, 42,
				0, 0, 0, 3,
				0, 0, 0, 39,
				0, 0, 0, 69,
				0, 0, 0, 20,
				0, 0, 0, 10,
				0, 0, 0, 0,
			},
			want: Computer{
				tree: map[uint32][]uint32{
					42: {39, 69, 20},
					10: {},
				},
				statsMap: map[uint32]Stats{
					42: {9, 8, 4},
					10: {2, 4, 5},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Deserialize(bytes.NewBuffer(tt.bytes))
			if err != nil {
				t.Errorf("Unexpected err: %v", err)
			}

			if reflect.DeepEqual(got, tt.want) {
				t.Logf("Output didnt match.\nExpected:\n%v\nGot:\n%v", tt.want, got)
			}
		})
	}
}

func TestWalk(t *testing.T) {
	// total parents: 4520 total nodes: 5478
	// X = 131184, O = 77904, tied = 46080, total = 255168
	c := Walk()

	expectedStats := Stats{X: 131184, O: 77904, T: 46080}
	stats := c.StatsMap()
	if stats[0] != expectedStats {
		t.Errorf("Mismatch stats\nExpected: %v\nGot: %v", expectedStats, stats[0])
	}

	if len(c.Tree()) != 4520 {
		t.Errorf("Mismatch tree size\nExpected: %v\nGot: %v", len(c.Tree()), 4520)
	}

	if len(c.StatsMap()) != 5478 {
		t.Errorf("Mismatch state map size\nExpected: %v\nGot: %v", len(c.StatsMap()), 5478)
	}
}

func TestVectorToBoard(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		seq    [3]rune
		want   uint32
		errStr string
	}{
		{
			name: "ez",
			s:    "010120102",
			seq:  [3]rune{'0', '1', '2'},
			want: 0b100001001001000100,
		},
		{
			name: "ez unicode",
			s:    " ⭕❌ ⭕❌ ⭕❌",
			seq:  [3]rune{' ', '❌', '⭕'},
			want: 0b011000011000011000,
		},
		{
			name:   "too big",
			s:      "0123456789",
			errStr: "length must be 9",
		},
		{
			name:   "too small",
			s:      "120",
			errStr: "length must be 9",
		},
		{
			name:   "illegal char",
			s:      "010120103",
			seq:    [3]rune{'0', '1', '2'},
			errStr: "illegal character",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := VectorToBoard(tt.s, tt.seq)
			if tt.errStr != "" {
				if err == nil {
					t.Errorf("expected error")
				} else if !strings.Contains(err.Error(), tt.errStr) {
					t.Errorf("expected error: %v, but got: %v", tt.errStr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.want != got {
				t.Errorf("mismatch\nputSlot() = %032b,\nwant\t%032b", got, tt.want)
			}
		})
	}
}

func TestBoardToVector(t *testing.T) {
	tests := []struct {
		name  string
		board uint32
		seq   [3]rune
		want  string
	}{
		{
			name:  "ez",
			board: 0b100100,
			seq:   [3]rune{'0', '1', '2'},
			want:  "012000000",
		},
		{
			name:  "ez unicode",
			board: 0b011000011000011000,
			seq:   [3]rune{' ', '❌', '⭕'},
			want:  " ⭕❌ ⭕❌ ⭕❌",
		},
		{
			name:  "upper bits are ignored",
			board: 0b10101010101001_100000000000100100,
			seq:   [3]rune{'0', '1', '2'},
			want:  "012000002",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BoardToVector(tt.board, tt.seq)
			if tt.want != got {
				t.Errorf("mismatch\nputSlot() = %s,\nwant\t%s", got, tt.want)
			}
		})
	}
}

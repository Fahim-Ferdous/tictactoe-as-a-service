package main

import (
	"encoding/json"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"unicode/utf8"

	computerPkg "tictoc/computer"
)

var computer = computerPkg.Walk()

func respondHelpMessage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(map[string]string{
		"mesg": "Use query parameter `board` to describe board, with the first character " +
			"being the top left corner. Use query parameter `pieces` to describe " +
			"\"empty\", \"cross\", \"circle\"",
	})
	if err != nil {
		slog.Error("write response", slog.Any("err", err))
		return
	}
}

func respondError(w http.ResponseWriter, _ *http.Request, err error) {
	w.Header().Add("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]string{
		"error": err.Error(),
	})
	if err != nil {
		slog.Error("write response", slog.Any("err", err))
	}
}

func respond(w http.ResponseWriter, _ *http.Request, resp any) {
	w.Header().Add("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		slog.Error("write response", slog.Any("err", err))
	}
}

func parsePieces(_ http.ResponseWriter, r *http.Request) [3]rune {
	pieces := [3]rune{'0', '1', '2'}

	rawPieces := r.URL.Query().Get("pieces")
	if utf8.RuneCountInString(rawPieces) == 3 {
		pieces = [3]rune([]rune(rawPieces))
	}

	return pieces
}

func parseBoard(w http.ResponseWriter, r *http.Request) (uint32, bool) {
	return parseBoardWithPieces(w, r, parsePieces(w, r))
}

func parseBoardWithPieces(w http.ResponseWriter, r *http.Request, pieces [3]rune) (uint32, bool) {
	if !r.URL.Query().Has("board") {
		respondHelpMessage(w, r)
		return 0, false
	}

	rawBoard := r.URL.Query().Get("board")
	board, err := computerPkg.VectorToBoard(rawBoard, pieces)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		respondError(w, r, err)
		return 0, false
	}

	return board, true
}

type NextResponse struct {
	Board string             `json:"board"`
	Stats *computerPkg.Stats `json:"stats,omitempty"`
}

func GCD[E uint32](a, b E) E {
	for b != 0 {
		a, b = b, a%b
	}

	return a
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/stats", func(w http.ResponseWriter, r *http.Request) {
		board, ok := parseBoard(w, r)
		if !ok {
			return
		}

		// Max moves min moves?
		stats := computer.Stats(board)
		respond(w, r, stats)
	})

	mux.HandleFunc("GET /v1/next", func(w http.ResponseWriter, r *http.Request) {
		pieces := parsePieces(w, r)
		board, ok := parseBoardWithPieces(w, r, pieces)
		if !ok {
			return
		}

		nextBoards := computer.Next(board)
		response := make([]NextResponse, len(nextBoards))
		for i := range nextBoards {
			response[i].Board = computerPkg.BoardToVector(nextBoards[i], pieces)
			if r.URL.Query().Has("stats") {
				stats := computer.Stats(nextBoards[i])
				response[i].Stats = &stats
			}
		}
		respond(w, r, response)
	})

	mux.HandleFunc("GET /v1/move", func(w http.ResponseWriter, r *http.Request) {
		pieces := parsePieces(w, r)
		board, ok := parseBoardWithPieces(w, r, pieces)
		if !ok {
			return
		}

		nextBoards := computer.Next(board)
		nextStats := make([]computerPkg.Stats, len(nextBoards))
		for i := range nextBoards {
			nextStats[i] = computer.Stats(nextBoards[i])
		}

		best := make([]int, 1, 9)
		crossTurn := computerPkg.WhichTurn(board)%2 == 0

		for i := 1; i < len(nextBoards); i++ {
			b := nextStats[best[0]]
			n := nextStats[i]

			bd := (b.X + b.O + b.T)
			nd := (n.X + n.O + n.T)

			lcm := bd * nd / GCD(bd, nd)

			b.X = b.X * (lcm / bd)
			b.O = b.O * (lcm / bd)
			b.T = b.T * (lcm / bd)

			n.X = n.X * (lcm / nd)
			n.O = n.O * (lcm / nd)
			n.T = n.T * (lcm / nd)

			// TODO: Compare to lose/tie?
			if (crossTurn && n.X > b.X) || (!crossTurn && n.O > b.O) {
				best = append(best[:0], i)
			} else if (crossTurn && n.X == b.X) || (!crossTurn && n.O == b.O) {
				best = append(best, i)
			}
		}

		picked := best[rand.IntN(len(best))]
		respond(w, r, NextResponse{
			Board: computerPkg.BoardToVector(nextBoards[picked], pieces),
			Stats: &nextStats[picked],
		})
	})

	srv := http.Server{
		Addr:    ":3000",
		Handler: mux,
	}

	if err := srv.ListenAndServe(); err != nil {
		slog.Error("srv.ListenAndSrv", slog.Any("err", err))
	}
}

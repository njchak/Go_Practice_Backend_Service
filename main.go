package main

import (
	"cmp"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"os"
	"slices"
	"strconv"
	"sync"
)

type Leaderboard interface {
	addToScore(username string, score int) error
	rank(username string) (int, error)
	topPlayers(n int) ([]LeaderBoardResponse, error)
}

type Server struct {
	players Leaderboard
}

type Player struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

type LeaderBoardResponse struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
	Rank  int    `json:"rank"`
}

type PlayerBase struct {
	mu   sync.RWMutex
	data map[string]*Player
}

func newPlayerBase() *PlayerBase {
	return &PlayerBase{
		data: make(map[string]*Player),
	}
}

func main() {

	seedFlag := flag.Bool("seed", false, "Includes example seeding data")
	flag.Parse()

	var serv Server

	serv.players = newPlayerBase()

	if *seedFlag {
		allPlayers := []*Player{
			{Name: "player 1", Score: 64},
			{Name: "player 2", Score: 20},
			{Name: "player 3", Score: 25},
			{Name: "player 3", Score: 25},
			{Name: "player 4", Score: 3},
			{Name: "player 5", Score: 1},
			{Name: "player 5", Score: 99},
			{Name: "BillyB", Score: 2000},
			{Name: "Billy", Score: 2000},
			{Name: "The King", Score: 20000},
		}

		for _, p := range allPlayers {
			serv.players.addToScore(p.Name, p.Score)
		}
	}

	http.HandleFunc("POST /addToScore", serv.httpAddToScore)
	http.HandleFunc("GET /rank", serv.httpRank)
	http.HandleFunc("GET /leaderboard", serv.httpTopPlayers)

	port := os.Getenv("LISTEN_ADDR")
	if port == "" {
		port = "localhost:8080"
	}
	log.Printf("Listening on %s...", port)
	log.Fatal(http.ListenAndServe(port, nil))

}

func (pb *PlayerBase) addToScore(username string, scoreToAdd int) error {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	if username == "" {
		return errors.New("Username is blank")
	}

	if scoreToAdd <= 0 {
		return errors.New("Cannot have a negative or zero score")
	}
	player, ok := pb.data[username]

	if ok {
		player.Score += scoreToAdd

	} else {
		pb.data[username] = &Player{Name: username, Score: scoreToAdd}
	}

	return nil
}

func (s Server) httpAddToScore(w http.ResponseWriter, req *http.Request) {

	var data Player

	body, err := io.ReadAll(req.Body)

	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(body, &data)

	if err != nil {
		http.Error(w, "Failed to unmarshal body", http.StatusBadRequest)
		return
	}

	err = s.players.addToScore(data.Name, data.Score)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	io.WriteString(w, fmt.Sprintf("Successfully added score of %d for %s\n", data.Score, data.Name))

}

func (s Server) httpRank(w http.ResponseWriter, req *http.Request) {

	uri := req.URL.Query()

	user := uri.Get("username")
	if user == "" {
		http.Error(w, "Missing proper parameter \"username\" in query", http.StatusBadRequest)
		return
	}

	playerRank, err := s.players.rank(user)

	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, strconv.Itoa(playerRank))

}

func (s Server) httpTopPlayers(w http.ResponseWriter, req *http.Request) {

	uri := req.URL.Query()

	n := uri.Get("n")
	if n == "" {
		http.Error(w, "Missing proper parameter in query: n, or no number assigned", http.StatusBadRequest)
		return
	}

	topN, err := strconv.Atoi(n)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	players, err := s.players.topPlayers(topN)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	output, err := json.Marshal(players)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(output)

}

func (pb *PlayerBase) topPlayers(n int) ([]LeaderBoardResponse, error) {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	if n <= 0 {
		return nil, errors.New("n must be a positive integer greater than or equal to 1")
	}

	if n > len(pb.data) {
		n = len(pb.data)
	}

	players := slices.Collect(maps.Values(pb.data))
	sortSlice(players)

	currRank := 1
	var prevScore int

	result := make([]LeaderBoardResponse, n)

	for i := 0; i < n; i++ {
		currPlayer := *players[i]
		if i != 0 {
			if prevScore != currPlayer.Score {
				currRank = i + 1
			}
		}

		prevScore = currPlayer.Score
		result[i] = LeaderBoardResponse{Name: currPlayer.Name, Score: currPlayer.Score, Rank: currRank}
		// If there is a tie in the last position, we still only return n players. This will be noted in documentation.
	}

	return result, nil
}

func (pb *PlayerBase) rank(username string) (int, error) {

	pb.mu.RLock()
	defer pb.mu.RUnlock()

	if username == "" {
		return -1, errors.New("Blank username entered, please enter a valid username")
	}
	player, ok := pb.data[username]

	if !ok {
		return -1, errors.New("Player does not exist")
	} else {

		players := slices.Collect(maps.Values(pb.data))

		sortSlice(players)

		currRank := 1
		var prevScore int
		for i, currPlayer := range players {

			if i != 0 {
				if prevScore != currPlayer.Score {
					currRank = i + 1
				}
			}

			if currPlayer.Name == player.Name {
				return currRank, nil
			}

			prevScore = currPlayer.Score
		}

		return currRank, nil
	}
}

func sortSlice(playerList []*Player) {
	slices.SortFunc(playerList, func(i, j *Player) int {
		return cmp.Or(cmp.Compare(j.Score, i.Score),
			cmp.Compare(i.Name, j.Name))
	})
}

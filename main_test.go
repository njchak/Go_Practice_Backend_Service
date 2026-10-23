package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestRank(t *testing.T) {
	lb := newPlayerBase()
	lb.addToScore("player 1", 64)
	lb.addToScore("player 2", 20)
	lb.addToScore("player 3", 25)
	lb.addToScore("player 3", 25)
	lb.addToScore("player 4", 3)
	lb.addToScore("player 5", 1)
	lb.addToScore("player 5", 99)
	lb.addToScore("Billy", 2000)
	lb.addToScore("BillyB", 2000)
	lb.addToScore("BillyC", 2500)

	want := 5
	msg, err := lb.rank("player 1")
	if want != msg || err != nil {
		t.Errorf(`Rank = %d, %v, want match for %d, nil`, msg, err, want)
	}
	want = 2
	msg, err = lb.rank("Billy")
	if want != msg || err != nil {
		t.Errorf(`Rank = %d, %v, want match for %d, nil`, msg, err, want)
	}
	want = 2
	msg, err = lb.rank("BillyB")
	if want != msg || err != nil {
		t.Errorf(`Rank = %d, %v, want match for %d, nil`, msg, err, want)
	}
	want = 4
	msg, err = lb.rank("player 5")
	if want != msg || err != nil {
		t.Errorf(`Rank = %d, %v, want match for %d, nil`, msg, err, want)
	}

	want = 1
	msg, err = lb.rank("BillyC")
	if want != msg || err != nil {
		t.Errorf(`Rank = %d, %v, want match for %d, nil`, msg, err, want)
	}

	msg, err = lb.rank("The King")
	if err == nil {
		t.Errorf(`Expected error`)
	}
	msg, err = lb.rank("")
	if err == nil {
		t.Errorf(`Expected error`)
	}
}

func TestTopPlayers(t *testing.T) {
	lb := newPlayerBase()
	lb.addToScore("player 1", 64)
	lb.addToScore("player 2", 20)
	lb.addToScore("player 3", 25)
	lb.addToScore("player 3", 25)
	lb.addToScore("player 4", 3)
	lb.addToScore("player 5", 1)
	lb.addToScore("player 5", 99)
	lb.addToScore("Billy", 2000)
	lb.addToScore("BillyB", 2000)
	lb.addToScore("BillyC", 2500)

	want := []LeaderBoardResponse{
		{Name: "BillyC", Score: 2500, Rank: 1},
		{Name: "Billy", Score: 2000, Rank: 2},
		{Name: "BillyB", Score: 2000, Rank: 2},
	}
	players, err := lb.topPlayers(3)
	if !slices.Equal(want, players) || err != nil {
		t.Errorf(`Top Players = %+v, %+v, want match for %+v, nil`, players, err, want)
	}

	want = []LeaderBoardResponse{
		{Name: "BillyC", Score: 2500, Rank: 1},
		{Name: "Billy", Score: 2000, Rank: 2},
		{Name: "BillyB", Score: 2000, Rank: 2},
		{Name: "player 5", Score: 100, Rank: 4},
		{Name: "player 1", Score: 64, Rank: 5},
		{Name: "player 3", Score: 50, Rank: 6},
		{Name: "player 2", Score: 20, Rank: 7},
		{Name: "player 4", Score: 3, Rank: 8},
	}
	players, err = lb.topPlayers(100)
	if !slices.Equal(want, players) || err != nil {
		t.Errorf(`Top Players = %+v, %+v, want match for %+v, nil`, players, err, want)
	}

	players, err = lb.topPlayers(-1)
	if err == nil {
		t.Errorf(`Expected error`)
	}

	players, err = lb.topPlayers(0)
	if err == nil {
		t.Errorf(`Expected error`)
	}
}

func TestAddScore(t *testing.T) {
	lb := newPlayerBase()
	lb.addToScore("player 1", 100)

	want := 100

	score := lb.data["player 1"].Score
	if want != score {
		t.Errorf(`Score = %d, expected: %d`, score, want)
	}

	lb.addToScore("player 1", 100)

	want = 200
	score = lb.data["player 1"].Score
	if want != score {
		t.Errorf(`Score = %d, expected: %d`, score, want)
	}

	err := lb.addToScore("player 1", 0)
	if err == nil {
		t.Errorf(`Expected error for adding 0 score`)
	}

	err = lb.addToScore("player 1", -1)
	if err == nil {
		t.Errorf(`Expected error for adding negative score`)
	}

}

func TestAddScoreHandler(t *testing.T) {

	var serv Server
	lb := newPlayerBase()
	serv.players = lb
	lb.addToScore(`Billy`, 10)

	testCases := []struct {
		name     string
		input    string
		expected int
		body     string
	}{
		{`New User`, `{"name": "Billy", "score" : 100}`, 200, "Successfully added score of 100 for Billy\n"},
		{`Malformed json`, `{{"name": "Billy", "score" : 100}`, 400, "Failed to unmarshal body\n"},
		{`empty username`, `{"name": "", "score" : 100}`, 400, "Username is blank\n"},
		{`Zero score`, `{"name": "Billy", "score" : 0}`, 400, "Cannot have a negative or zero score\n"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, `/addToScore`, strings.NewReader(tc.input))
			rr := httptest.NewRecorder()

			handler := http.HandlerFunc(serv.httpAddToScore)
			handler.ServeHTTP(rr, req)

			if rr.Code != tc.expected {
				t.Errorf("expected status %d, got %d", tc.expected, rr.Code)
			}

			if rr.Body.String() != tc.body {
				t.Errorf("expected body %q, got %q", tc.body, rr.Body.String())
			}
		})
	}
}

func TestRankHandler(t *testing.T) {

	var serv Server
	lb := newPlayerBase()
	serv.players = lb
	lb.addToScore(`Billy`, 10)

	testCases := []struct {
		name     string
		input    string
		expected int
		body     string
	}{
		{`Unregistered User`, `unregisteredUser`, 404, "Player does not exist\n"},
		{`Billy`, `Billy`, 200, "1"},
		{`empty string`, ``, 400, "Missing proper parameter \"username\" in query\n"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf(`/rank?username=%s`, tc.input), nil)
			rr := httptest.NewRecorder()

			handler := http.HandlerFunc(serv.httpRank)
			handler.ServeHTTP(rr, req)

			if rr.Code != tc.expected {
				t.Errorf("expected status %d, got %d", tc.expected, rr.Code)
			}

			if rr.Body.String() != tc.body {
				t.Errorf("expected body %q, got %q", tc.body, rr.Body.String())
			}
		})
	}
}

func TestLeaderboardHandler(t *testing.T) {

	var serv Server
	lb := newPlayerBase()
	serv.players = lb
	lb.addToScore(`Billy`, 10)
	lb.addToScore(`BillyB`, 10)
	lb.addToScore(`The King`, 100)

	testCases := []struct {
		name     string
		input    string
		expected int
		body     string
	}{
		{`Bigger than number of people in leaderboard`, `100`, 200,
			"[{\"name\":\"The King\",\"score\":100,\"rank\":1},{\"name\":\"Billy\",\"score\":10,\"rank\":2},{\"name\":\"BillyB\",\"score\":10,\"rank\":2}]"},
		{`Tie in last position`, `2`, 200, "[{\"name\":\"The King\",\"score\":100,\"rank\":1},{\"name\":\"Billy\",\"score\":10,\"rank\":2}]"},
		{`Negative n`, `-1`, 400, "n must be a positive integer greater than or equal to 1\n"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf(`/leaderboard?n=%s`, tc.input), nil)
			rr := httptest.NewRecorder()

			handler := http.HandlerFunc(serv.httpTopPlayers)
			handler.ServeHTTP(rr, req)

			if rr.Code != tc.expected {
				t.Errorf("expected status %d, got %d", tc.expected, rr.Code)
			}

			if rr.Body.String() != tc.body {
				t.Errorf("expected body %q, got %q", tc.body, rr.Body.String())
			}
		})
	}
}

func TestConcurrency(t *testing.T) {
	var wg sync.WaitGroup

	lb := newPlayerBase()

	lb.addToScore("Player 1", 100)
	for i := range 1000 {
		wg.Go(func() {
			lb.addToScore("Racer "+strconv.Itoa(i), 10)
		})

		wg.Go(func() {
			lb.addToScore("Player 1", 25)
		})

		wg.Go(func() {
			lb.rank("Player 1")
		})

		wg.Go(func() {
			lb.topPlayers(25)
		})

	}

	wg.Wait()

	if lb.data["Player 1"].Score != 25100 {
		t.Errorf("expected score %d, got %d", 25100, lb.data["Player 1"].Score)
	}
	if len(lb.data) != 1001 {
		t.Errorf("Number of players expected %d, got %d", 1001, len(lb.data))
	}

}

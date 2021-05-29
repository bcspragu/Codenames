package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bcspragu/Codenames/codenames"
	"github.com/bcspragu/Codenames/memdb"
	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
)

func TestBasicallyEverything(t *testing.T) {
	// This is a hodge-podge test that tests out the entire flow end-to-end,
	// because this is a personal project and I don't have the wherewithal to add
	// more modular tests.
	env := setup()

	for i := 0; i < 5; i++ {
		env.createUser(t, fmt.Sprintf("Test%d", i))
	}

	// Sanity check the auth works by requesting a users information back.
	gotUser := env.user(t, 3 /* user index 3 */)
	wantUser := &codenames.User{
		ID:   "user_3",
		Name: "Test3",
	}
	if diff := cmp.Diff(wantUser, gotUser); diff != "" {
		t.Errorf("unexpected user (-want +got)\n%s", diff)
	}

	gID := env.createGame(t, 1)
	gotGame, err := env.db.Game(gID)
	if err != nil {
		t.Fatalf("failed to load game %q: %v", gID, err)
	}
	wantGame := &codenames.Game{
		ID:        "game_0",
		CreatedBy: "user_1",
		Status:    codenames.Pending,
		State: &codenames.GameState{
			ActiveTeam:   codenames.BlueTeam,
			ActiveRole:   codenames.SpymasterRole,
			Board:        &codenames.Board{Cards: startingBoardCards()},
			StartingTeam: codenames.BlueTeam,
		},
	}
	if diff := cmp.Diff(wantGame, gotGame); diff != "" {
		t.Errorf("unexpected game (-want +got)\n%s", diff)
	}

	gotPendingGames := env.pendingGames(t)
	wantPendingGames := []codenames.GameID{"game_0"}
	if diff := cmp.Diff(wantPendingGames, gotPendingGames); diff != "" {
		t.Errorf("unexpected pending game IDs (-want +got)\n%s", diff)
	}

	// Have four players join that game.
	for i := 0; i < 4; i++ {
		env.joinGame(t, gID, i)
	}

	// Have the game creator assign roles.
	assignRole := func(idx int, role codenames.Role, team codenames.Team) {
		env.assignRole(t, gID, 1 /* creator index */, fmt.Sprintf("user_%d", idx), role, team)
	}

	assignRole(0, codenames.SpymasterRole, codenames.BlueTeam)
	assignRole(1, codenames.SpymasterRole, codenames.RedTeam)
	assignRole(2, codenames.OperativeRole, codenames.BlueTeam)
	assignRole(3, codenames.OperativeRole, codenames.RedTeam)

	// Have the game creator start the game.
	env.startGame(t, gID, 1)
}

// startingBoardCards returns the cards we expect on the test board, since we
// use a deterministic pseudo-random number generator.
func startingBoardCards() []codenames.Card {
	return []codenames.Card{
		{Codename: "dwarf", Agent: codenames.Bystander},
		{Codename: "green", Agent: codenames.Bystander},
		{Codename: "doctor", Agent: codenames.BlueAgent},
		{Codename: "ship", Agent: codenames.RedAgent},
		{Codename: "dance", Agent: codenames.Bystander},
		{Codename: "time", Agent: codenames.RedAgent},
		{Codename: "pool", Agent: codenames.BlueAgent},
		{Codename: "cover", Agent: codenames.Bystander},
		{Codename: "fighter", Agent: codenames.RedAgent},
		{Codename: "horse", Agent: codenames.RedAgent},
		{Codename: "strike", Agent: codenames.BlueAgent},
		{Codename: "cast", Agent: codenames.RedAgent},
		{Codename: "string", Agent: codenames.Bystander},
		{Codename: "greece", Agent: codenames.BlueAgent},
		{Codename: "fence", Agent: codenames.BlueAgent},
		{Codename: "drill", Agent: codenames.BlueAgent},
		{Codename: "button", Agent: codenames.Assassin},
		{Codename: "cycle", Agent: codenames.RedAgent},
		{Codename: "chest", Agent: codenames.RedAgent},
		{Codename: "pitch", Agent: codenames.Bystander},
		{Codename: "unicorn", Agent: codenames.BlueAgent},
		{Codename: "agent", Agent: codenames.BlueAgent},
		{Codename: "kiwi", Agent: codenames.Bystander},
		{Codename: "swing", Agent: codenames.RedAgent},
		{Codename: "skyscraper", Agent: codenames.BlueAgent},
	}
}

func (env *testEnv) createUser(t *testing.T, name string) {
	req := struct {
		Name string `json:"name"`
	}{name}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/user", toBody(t, req))
	if err := env.srv.serveCreateUser(w, r); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	auth := w.Header().Get("Set-Cookie")
	if auth == "" {
		t.Fatal("no auth was provided in create user response")
	}
	if !strings.HasPrefix(auth, "Authorization=") {
		t.Fatalf("malformed authorization cookie %q", auth)
	}
	env.userAuth = append(env.userAuth, strings.TrimPrefix(auth, "Authorization="))
}

func (env *testEnv) user(t *testing.T, authIdx int) *codenames.User {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/user", nil)
	env.addAuth(r, authIdx)

	if err := env.srv.serveUser(w, r); err != nil {
		t.Fatalf("failed to get user: %v", err)
	}

	var u codenames.User
	fromBody(t, w, &u)
	return &u
}

func (env *testEnv) createGame(t *testing.T, authIdx int) codenames.GameID {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/game", nil)
	env.addAuth(r, authIdx)

	if err := env.srv.serveCreateGame(w, r); err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	var resp struct {
		ID string `json:"id"`
	}
	fromBody(t, w, &resp)
	return codenames.GameID(resp.ID)
}

func (env *testEnv) pendingGames(t *testing.T) []codenames.GameID {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/games", nil)

	if err := env.srv.servePendingGames(w, r); err != nil {
		t.Fatalf("failed to get pending games: %v", err)
	}

	var resp []codenames.GameID
	fromBody(t, w, &resp)
	return resp
}

func (env *testEnv) joinGame(t *testing.T, gID codenames.GameID, authIdx int) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/game/"+string(gID)+"/join", nil)
	r = mux.SetURLVars(r, map[string]string{"id": string(gID)})
	env.addAuth(r, authIdx)

	handler := env.srv.requireGameAuth(env.srv.serveJoinGame, isGamePending())
	if err := handler(w, r); err != nil {
		t.Fatalf("failed to join game: %v", err)
	}
}

func (env *testEnv) assignRole(t *testing.T, gID codenames.GameID, authIdx int, userID string, role codenames.Role, team codenames.Team) {
	req := struct {
		UserID string `json:"user_id"`
		Team   string `json:"team"`
		Role   string `json:"role"`
	}{userID, string(team), string(role)}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/game/"+string(gID)+"/assignRole", toBody(t, req))
	r = mux.SetURLVars(r, map[string]string{"id": string(gID)})
	env.addAuth(r, authIdx)

	handler := env.srv.requireGameAuth(env.srv.serveAssignRole, isGameCreator(), isGamePending())
	if err := handler(w, r); err != nil {
		t.Fatalf("failed to assign role: %v", err)
	}
}

func (env *testEnv) startGame(t *testing.T, gID codenames.GameID, authIdx int) {
	req := struct {
		RandomAssignment bool `json:"random_assignment"`
	}{false}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/game/"+string(gID)+"/start", toBody(t, req))
	r = mux.SetURLVars(r, map[string]string{"id": string(gID)})
	env.addAuth(r, authIdx)

	handler := env.srv.requireGameAuth(env.srv.serveStartGame, isGameCreator(), isGamePending())
	if err := handler(w, r); err != nil {
		t.Fatalf("failed to start game: %v", err)
	}
}

func (env *testEnv) addAuth(r *http.Request, authIdx int) {
	r.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: env.userAuth[authIdx],
	})
}

func toBody(t *testing.T, body interface{}) io.Reader {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		t.Fatalf("failed to encode body: %v", err)
	}
	return &buf
}

func fromBody(t *testing.T, w *httptest.ResponseRecorder, resp interface{}) {
	if err := json.NewDecoder(w.Body).Decode(resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
}

type testEnv struct {
	db       *memdb.DB
	srv      *Srv
	userAuth []string
}

func setup() *testEnv {
	db := memdb.New()

	return &testEnv{
		db: db,
		srv: New(
			db,
			rand.New(rand.NewSource(0)),
			setupCookies(),
		),
	}
}

func setupCookies() *securecookie.SecureCookie {
	return securecookie.New(
		[]byte{
			1, 2, 3, 4, 5, 6, 7, 8,
			9, 10, 11, 12, 13, 14, 15, 16,
			17, 18, 19, 20, 21, 22, 23, 24,
			25, 26, 27, 28, 29, 30, 31, 32,
		},
		[]byte{
			33, 34, 35, 36, 37, 38, 39, 40,
			41, 42, 43, 44, 45, 46, 47, 48,
			49, 50, 51, 52, 53, 54, 55, 56,
			57, 58, 59, 60, 61, 62, 63, 64,
		})
}

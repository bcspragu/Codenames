package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"

	"github.com/bcspragu/Codenames/codenames"
	"github.com/bcspragu/Codenames/web"
)

type client struct {
	scheme string
	addr   string
	client *http.Client
}

func newClient(scheme, addr string) (*client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %v", err)
	}

	return &client{
		scheme: scheme,
		addr:   addr,
		client: &http.Client{Jar: jar},
	}, nil
}

func (c *client) createUser(name string) (string, error) {
	body := struct {
		Name string `json:"name"`
	}{name}

	req, err := http.NewRequest(http.MethodPost, c.scheme+"://"+c.addr+"/api/user", toBody(body))
	if err != nil {
		return "", fmt.Errorf("failed to form request: %w", err)
	}

	var resp struct {
		UserID string `json:"user_id"`
	}
	if err := c.do(req, &resp); err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}
	return resp.UserID, nil
}

func (c *client) createGame() (string, error) {
	req, err := http.NewRequest(http.MethodPost, c.scheme+"://"+c.addr+"/api/game", nil)
	if err != nil {
		return "", fmt.Errorf("failed to form request: %w", err)
	}

	var resp struct {
		ID string `json:"id"`
	}
	if err := c.do(req, &resp); err != nil {
		return "", fmt.Errorf("failed to create game: %w", err)
	}
	return resp.ID, nil
}

func (c *client) players(gID string) ([]*web.Player, error) {
	req, err := http.NewRequest(http.MethodGet, c.scheme+"://"+c.addr+"/api/game/"+gID+"/players", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to form request: %w", err)
	}

	var resp []*web.Player
	if err := c.do(req, &resp); err != nil {
		return nil, fmt.Errorf("failed to load players: %w", err)
	}
	return resp, nil
}

func (c *client) joinGame(gID string) error {
	req, err := http.NewRequest(http.MethodPost, c.scheme+"://"+c.addr+"/api/game/"+gID+"/join", nil)
	if err != nil {
		return fmt.Errorf("failed to form request: %w", err)
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("failed to join game: %w", err)
	}

	return nil
}

func (c *client) assignRole(gID, uID, team, role string) error {
	body := struct {
		UserID string `json:"user_id"`
		Team   string `json:"team"`
		Role   string `json:"role"`
	}{uID, team, role}

	req, err := http.NewRequest(http.MethodPost, c.scheme+"://"+c.addr+"/api/game/"+gID+"/assignRole", toBody(body))
	if err != nil {
		return fmt.Errorf("failed to form request: %w", err)
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	return nil
}

func (c *client) startGame(gID string) error {
	body := struct {
		RandomAssignment bool `json:"random_assignment"`
	}{true}

	req, err := http.NewRequest(http.MethodPost, c.scheme+"://"+c.addr+"/api/game/"+gID+"/start", toBody(body))
	if err != nil {
		return fmt.Errorf("failed to form request: %w", err)
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("failed to start game: %w", err)
	}

	return nil
}

func (c *client) giveClue(gID string, clue *codenames.Clue) error {
	body := struct {
		Word  string `json:"word"`
		Count int    `json:"count"`
	}{clue.Word, clue.Count}

	req, err := http.NewRequest(http.MethodPost, c.scheme+"://"+c.addr+"/api/game/"+gID+"/clue", toBody(body))
	if err != nil {
		return fmt.Errorf("failed to form request: %w", err)
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("failed to give clue to game: %w", err)
	}

	return nil
}

func (c *client) giveGuess(gID, guess string, confirmed bool) error {
	body := struct {
		Guess     string `json:"guess"`
		Confirmed bool   `json:"confirmed"`
	}{guess, confirmed}

	req, err := http.NewRequest(http.MethodPost, c.scheme+"://"+c.addr+"/api/game/"+gID+"/guess", toBody(body))
	if err != nil {
		return fmt.Errorf("failed to form request: %w", err)
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("failed to give clue to game: %w", err)
	}

	return nil
}

func (c *client) do(req *http.Request, resp interface{}) error {
	httpResp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		return handleError(httpResp)
	}

	if resp != nil {
		if err := json.NewDecoder(httpResp.Body).Decode(resp); err != nil {
			return fmt.Errorf("failed to decode response body: %w", err)
		}
	}

	return nil
}

type httpError struct {
	statusCode int
	body       string
	err        error
}

func (h *httpError) Error() string {
	if h.err != nil {
		return fmt.Sprintf("[%d] failed to handle error: %v", h.statusCode, h.err)
	}
	return fmt.Sprintf("[%d] error from server: %s", h.statusCode, h.body)
}

func handleError(resp *http.Response) error {
	dat, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &httpError{
			statusCode: resp.StatusCode,
			err:        fmt.Errorf("failed to read error response body: %w", err),
		}
	}

	return &httpError{
		statusCode: resp.StatusCode,
		body:       string(dat),
	}
}

func toBody(req interface{}) io.Reader {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return &errReader{err: err}
	}
	return &buf
}

type errReader struct {
	err error
}

func (e *errReader) Read(_ []byte) (int, error) {
	return 0, e.err
}

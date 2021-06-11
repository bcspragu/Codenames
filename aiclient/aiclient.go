// Package aiclient implements the simple interface for communicating with the
// AI service, mainly saying 'hey, join this game as an AI'.
package aiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/bcspragu/Codenames/codenames"
)

type Client struct {
	secret string
	scheme string
	addr   string
	http   *http.Client
}

func New(secret, scheme, addr string) *Client {
	return &Client{
		secret: secret,
		scheme: scheme,
		addr:   addr,
		http:   http.DefaultClient,
	}
}

func (c *Client) JoinGame(gID codenames.GameID) (codenames.RobotID, error) {
	body := struct {
		GameID string `json:"game_id"`
	}{string(gID)}

	endpoint := c.scheme + "://" + c.addr + "/join"
	req, err := http.NewRequest(http.MethodPost, endpoint, toBody(body))
	if err != nil {
		return "", fmt.Errorf("failed to form request: %w", err)
	}
	req.Header.Set("Authorization", c.secret)

	var resp struct {
		RobotID string `json:"robot_id"`
		Success bool   `json:"success"`
	}
	if err := c.do(req, &resp); err != nil {
		return "", fmt.Errorf("failed to request AI join a game: %w", err)
	}
	return codenames.RobotID(resp.RobotID), nil
}

func (c *Client) do(req *http.Request, resp interface{}) error {
	httpResp, err := c.http.Do(req)
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

func toBody(req interface{}) io.Reader {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return &errReader{err: err}
	}
	return &buf
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

type errReader struct {
	err error
}

func (e *errReader) Read(_ []byte) (int, error) {
	return 0, e.err
}

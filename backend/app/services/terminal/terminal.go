package terminal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"

	contractshttp "github.com/goravel/framework/contracts/http"

	"ebox/app/services/docker"
	"ebox/app/services/settings"
)

const (
	idleTimeout  = 30 * time.Minute
	pingInterval = 20 * time.Second
	writeTimeout = 10 * time.Second
	readLimit    = 1 << 20
	outputBuffer = 32 * 1024
)

type Service struct{}

var defaultService = &Service{}

func Default() *Service {
	return defaultService
}

type controlMessage struct {
	Type string `json:"type"`
	Cols uint   `json:"cols"`
	Rows uint   `json:"rows"`
	Data string `json:"data"`
}

type session struct {
	kind   string
	pty    *os.File
	exec   *docker.ExecSession
	reader interface {
		Read([]byte) (int, error)
	}
	writer interface {
		Write([]byte) (int, error)
	}
	resize func(width, height uint) error
	close  func()
}

// Target describes a terminal target shown in the UI.
type Target struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	State    string `json:"state"`
	Image    string `json:"image,omitempty"`
	Resource string `json:"resource,omitempty"`
}

// Targets lists the host plus every managed container.
func (s *Service) Targets(ctx context.Context) ([]Target, error) {
	targets := []Target{
		{ID: "host", Name: "Host (panel)", Kind: "host", State: "running"},
	}

	containers, err := docker.Default().List(ctx, docker.Filters{}.Add("label", docker.LabelManaged+"=true"))
	if err != nil {
		return targets, err
	}

	for _, container := range containers {
		targets = append(targets, Target{
			ID:       container.ID,
			Name:     container.Name,
			Kind:     "container",
			State:    container.State,
			Image:    container.Image,
			Resource: container.Labels[docker.LabelResourceType],
		})
	}

	return targets, nil
}

// Connect upgrades the request to a WebSocket and bridges it to a shell.
func (s *Service) Connect(ctx contractshttp.Context) error {
	request := ctx.Request().Origin()

	upgrader := websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin:     originAllowed,
	}

	connection, err := upgrader.Upgrade(ctx.Response().Writer(), request, nil)
	if err != nil {
		return err
	}
	defer connection.Close()

	query := request.URL.Query()
	target := query.Get("target")
	if target == "" {
		target = "host"
	}

	cols := clampSize(parseUint(query.Get("cols")), 80, 20, 500)
	rows := clampSize(parseUint(query.Get("rows")), 24, 5, 300)

	session, err := s.start(request.Context(), target, cols, rows)
	if err != nil {
		_ = connection.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[31m"+err.Error()+"\x1b[0m\r\n\r\n"))

		return err
	}
	defer session.close()

	return bridge(connection, session)
}

func (s *Service) start(ctx context.Context, target string, cols, rows uint) (*session, error) {
	if target == "host" {
		return startHost(cols, rows)
	}

	inspect, err := docker.Default().Inspect(ctx, target)
	if err != nil {
		return nil, errors.New("terminal: the container is not available")
	}

	if inspect.Config == nil || inspect.Config.Labels[docker.LabelManaged] != "true" {
		return nil, errors.New("terminal: only containers managed by ebox can be opened")
	}

	if inspect.State == nil || !inspect.State.Running {
		return nil, errors.New("terminal: the container is not running")
	}

	shell := []string{"/bin/sh", "-c", "if command -v bash >/dev/null 2>&1; then exec bash; else exec sh; fi"}

	execSession, err := docker.Default().Exec(ctx, inspect.ID, shell, true, cols, rows)
	if err != nil {
		return nil, err
	}

	return &session{
		kind:   "container",
		exec:   execSession,
		reader: execSession.Response.Reader,
		writer: execSession.Response.Conn,
		resize: func(width, height uint) error {
			return execSession.Resize(context.Background(), width, height)
		},
		close: execSession.Close,
	}, nil
}

func startHost(cols, rows uint) (*session, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	if _, err := os.Stat(shell); err != nil {
		shell = "/bin/sh"
	}

	command := exec.Command(shell, "-l")
	command.Env = append(os.Environ(), "TERM=xterm-256color")

	file, err := pty.StartWithSize(command, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return nil, fmt.Errorf("terminal: could not start a shell: %w", err)
	}

	return &session{
		kind:   "host",
		pty:    file,
		reader: file,
		writer: file,
		resize: func(width, height uint) error {
			return pty.Setsize(file, &pty.Winsize{Cols: uint16(width), Rows: uint16(height)})
		},
		close: func() {
			_ = file.Close()
			if command.Process != nil {
				_ = command.Process.Kill()
			}
			_, _ = command.Process.Wait()
		},
	}, nil
}

func bridge(connection *websocket.Conn, session *session) error {
	var (
		writeMu sync.Mutex
		done    = make(chan struct{})
		once    sync.Once
	)

	finish := func() {
		once.Do(func() { close(done) })
	}
	defer finish()

	write := func(messageType int, payload []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()

		_ = connection.SetWriteDeadline(time.Now().Add(writeTimeout))

		return connection.WriteMessage(messageType, payload)
	}

	// process → websocket
	go func() {
		defer finish()

		buffer := make([]byte, outputBuffer)
		for {
			read, err := session.reader.Read(buffer)
			if read > 0 {
				if writeErr := write(websocket.BinaryMessage, buffer[:read]); writeErr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// keepalive
	go func() {
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := write(websocket.PingMessage, nil); err != nil {
					finish()

					return
				}
			}
		}
	}()

	connection.SetReadLimit(readLimit)
	_ = connection.SetReadDeadline(time.Now().Add(idleTimeout))
	connection.SetPongHandler(func(string) error {
		return connection.SetReadDeadline(time.Now().Add(idleTimeout))
	})

	for {
		select {
		case <-done:
			return nil
		default:
		}

		messageType, payload, err := connection.ReadMessage()
		if err != nil {
			return nil
		}

		_ = connection.SetReadDeadline(time.Now().Add(idleTimeout))

		if messageType == websocket.TextMessage {
			var message controlMessage
			if err := json.Unmarshal(payload, &message); err != nil {
				continue
			}

			switch message.Type {
			case "resize":
				if message.Cols > 0 && message.Rows > 0 {
					_ = session.resize(
						clampSize(message.Cols, 80, 20, 500),
						clampSize(message.Rows, 24, 5, 300),
					)
				}
			case "input":
				_, _ = session.writer.Write([]byte(message.Data))
			}

			continue
		}

		if _, err := session.writer.Write(payload); err != nil {
			return nil
		}
	}
}

func originAllowed(request *http.Request) bool {
	origin := request.Header.Get("Origin")
	if origin == "" {
		return true
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}

	if strings.EqualFold(parsed.Host, request.Host) {
		return true
	}

	if domain := settings.Default().Get("panel_domain", ""); domain != "" {
		if strings.EqualFold(parsed.Host, domain) || strings.EqualFold(parsed.Hostname(), domain) {
			return true
		}
	}

	host := strings.ToLower(parsed.Hostname())

	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func parseUint(value string) uint {
	var parsed uint
	for _, character := range value {
		if character < '0' || character > '9' {
			return 0
		}
		parsed = parsed*10 + uint(character-'0')
	}

	return parsed
}

func clampSize(value, fallback, minimum, maximum uint) uint {
	if value < minimum || value > maximum {
		if value < minimum {
			return fallback
		}

		return maximum
	}

	return value
}

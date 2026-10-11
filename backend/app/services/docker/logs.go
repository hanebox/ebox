package docker

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
)

const maxPartialLine = 256 * 1024

// LineWriter turns a byte stream into complete lines.
type LineWriter struct {
	mu     sync.Mutex
	buffer []byte
	sink   func(string)
}

func NewLineWriter(sink func(string)) *LineWriter {
	return &LineWriter{sink: sink}
}

func (w *LineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buffer = append(w.buffer, p...)

	for {
		index := bytes.IndexByte(w.buffer, '\n')
		if index < 0 {
			break
		}

		line := strings.TrimRight(string(w.buffer[:index]), "\r")
		w.sink(line)
		w.buffer = w.buffer[index+1:]
	}

	if len(w.buffer) > maxPartialLine {
		w.sink(string(w.buffer))
		w.buffer = nil
	}

	return len(p), nil
}

// Flush emits any partial line that has not been terminated yet.
func (w *LineWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.buffer) == 0 {
		return
	}

	line := strings.TrimRight(string(w.buffer), "\r")
	w.buffer = nil

	if line != "" {
		w.sink(line)
	}
}

// StreamLogLines follows a container's log, emitting complete lines.
func (s *Service) StreamLogLines(ctx context.Context, id string, tail int, follow bool, sink func(string)) error {
	cli, err := s.handle()
	if err != nil {
		return err
	}

	tailValue := "all"
	if tail > 0 {
		tailValue = itoa(tail)
	}

	reader, err := cli.ContainerLogs(ctx, id, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Tail:       tailValue,
	})
	if err != nil {
		return errors.New(FriendlyError(err))
	}
	defer reader.Close()

	inspect, err := cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return errors.New(FriendlyError(err))
	}

	writer := NewLineWriter(sink)

	if inspect.Container.Config != nil && inspect.Container.Config.Tty {
		_, err = io.Copy(writer, reader)
	} else {
		_, err = stdcopy.StdCopy(writer, writer, reader)
	}

	writer.Flush()

	if err != nil && ctx.Err() == nil && !isClosedError(err) {
		return err
	}

	return nil
}

func isClosedError(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		return true
	}

	message := err.Error()

	return strings.Contains(message, "use of closed") || strings.Contains(message, "context canceled")
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}

	negative := value < 0
	if negative {
		value = -value
	}

	var buffer [20]byte
	index := len(buffer)

	for value > 0 {
		index--
		buffer[index] = byte('0' + value%10)
		value /= 10
	}

	if negative {
		index--
		buffer[index] = '-'
	}

	return string(buffer[index:])
}

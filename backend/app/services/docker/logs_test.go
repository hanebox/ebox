package docker

import (
	"strings"
	"testing"

	"github.com/moby/moby/api/types/events"
	"github.com/stretchr/testify/require"
)

func TestLineWriterSplitsLines(t *testing.T) {
	lines := []string{}
	writer := NewLineWriter(func(line string) { lines = append(lines, line) })

	_, err := writer.Write([]byte("first\nsecond\nthi"))
	require.NoError(t, err)
	require.Equal(t, []string{"first", "second"}, lines)

	_, err = writer.Write([]byte("rd\n"))
	require.NoError(t, err)
	require.Equal(t, []string{"first", "second", "third"}, lines)

	writer.Flush()
	require.Equal(t, []string{"first", "second", "third"}, lines)
}

func TestLineWriterFlushesPartialLine(t *testing.T) {
	lines := []string{}
	writer := NewLineWriter(func(line string) { lines = append(lines, line) })

	_, err := writer.Write([]byte("no trailing newline"))
	require.NoError(t, err)
	require.Empty(t, lines)

	writer.Flush()
	require.Equal(t, []string{"no trailing newline"}, lines)
}

func TestLineWriterStripsCarriageReturns(t *testing.T) {
	lines := []string{}
	writer := NewLineWriter(func(line string) { lines = append(lines, line) })

	_, err := writer.Write([]byte("windows\r\nstyle\r\n"))
	require.NoError(t, err)
	require.Equal(t, []string{"windows", "style"}, lines)
}

func TestLineWriterKeepsEmptyLines(t *testing.T) {
	lines := []string{}
	writer := NewLineWriter(func(line string) { lines = append(lines, line) })

	_, err := writer.Write([]byte("a\n\nb\n"))
	require.NoError(t, err)
	require.Equal(t, []string{"a", "", "b"}, lines)
}

func TestEventActionMapping(t *testing.T) {
	cases := map[events.Action]string{
		events.ActionStart:                 "start",
		events.ActionStop:                  "stop",
		events.ActionDie:                   "die",
		events.ActionHealthStatusHealthy:   "healthy",
		events.ActionHealthStatusUnhealthy: "unhealthy",
		events.Action("image.pull"):        "",
	}

	for action, expected := range cases {
		mapped, ok := EventAction(action)
		require.Equal(t, expected, mapped, "action %s", action)
		require.Equal(t, expected != "", ok, "action %s", action)
	}
}

func TestItoa(t *testing.T) {
	require.Equal(t, "0", itoa(0))
	require.Equal(t, "42", itoa(42))
	require.Equal(t, "-7", itoa(-7))
	require.Equal(t, "123456789", itoa(123456789))
}

func TestIsClosedError(t *testing.T) {
	require.True(t, isClosedError(errString("use of closed network connection")))
	require.True(t, isClosedError(errString("context canceled")))
	require.False(t, isClosedError(errString(strings.Repeat("boom", 3))))

}

package llmclient

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
)

var testLogger zerolog.Logger

func TestMain(m *testing.M) {
	logLevel := zerolog.InfoLevel
	if _, ok := os.LookupEnv("DEBUG"); ok {
		logLevel = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(logLevel)

	// For demonstration, we'll just send everything to stderr in console format.
	// If you prefer to capture logs in each test, see the next snippet below.
	consoleWriter := zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = os.Stderr
	})

	testLogger = zerolog.New(consoleWriter).Level(logLevel).With().Timestamp().Logger()

	// Now run the tests
	code := m.Run()

	os.Exit(code)
}

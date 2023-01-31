package poc

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

var _EMPTY_ = ""

func require_True(t *testing.T, b bool) {
	t.Helper()
	if !b {
		t.Fatalf("require true, but got false")
	}
}

func require_False(t *testing.T, b bool) {
	t.Helper()
	if b {
		t.Fatalf("require false, but got true")
	}
}

func require_NoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("require no error, but got: %v", err)
	}
}

func require_Contains(t *testing.T, s string, subStrs ...string) {
	t.Helper()
	for _, subStr := range subStrs {
		if !strings.Contains(s, subStr) {
			t.Fatalf("require %q to be contained in %q", subStr, s)
		}
	}
}

func require_Error(t *testing.T, err error, expected ...error) {
	t.Helper()
	if err == nil {
		t.Fatalf("require error, but got none")
	}
	if len(expected) == 0 {
		return
	}
	// Try to strip nats prefix from Go library if present.
	const natsErrPre = "nats: "
	eStr := err.Error()
	if strings.HasPrefix(eStr, natsErrPre) {
		eStr = strings.Replace(eStr, natsErrPre, _EMPTY_, 1)
	}

	for _, e := range expected {
		if err == e || strings.Contains(eStr, e.Error()) || strings.Contains(e.Error(), eStr) {
			return
		}
	}
	t.Fatalf("Expected one of %v, got '%v'", expected, err)
}

func require_Equal(t *testing.T, a, b string) {
	t.Helper()
	if strings.Compare(a, b) != 0 {
		t.Fatalf("require equal, but got: %v != %v", a, b)
	}
}

func require_NotEqual(t *testing.T, a, b [32]byte) {
	t.Helper()
	if bytes.Equal(a[:], b[:]) {
		t.Fatalf("require not equal, but got: %v != %v", a, b)
	}
}

func require_Len(t *testing.T, a, b int) {
	t.Helper()
	if a != b {
		t.Fatalf("require len, but got: %v != %v", a, b)
	}
}

func createTempFile(t testing.TB, prefix string) *os.File {
	t.Helper()
	tempDir := t.TempDir()
	f, err := os.CreateTemp(tempDir, prefix)
	require_NoError(t, err)
	return f
}

func createConfFile(t testing.TB, content []byte) string {
	t.Helper()
	conf := createTempFile(t, _EMPTY_)
	fName := conf.Name()
	conf.Close()
	if err := os.WriteFile(fName, content, 0666); err != nil {
		t.Fatalf("Error writing conf file: %v", err)
	}
	return fName
}

func DefaultOptions() *server.Options {
	return &server.Options{
		Host:     "127.0.0.1",
		Port:     -1,
		HTTPPort: -1,
		Cluster:  server.ClusterOpts{Port: -1, Name: "abc"},
		NoLog:    true,
		NoSigs:   true,
		Debug:    true,
		Trace:    true,
	}
}

// RunServer starts a new Go Routine based server
func RunServer(opts *server.Options) *server.Server {
	if opts == nil {
		opts = DefaultOptions()
	}
	s, err := server.NewServer(opts)
	if err != nil || s == nil {
		panic(fmt.Sprintf("No NATS Server object returned: %v", err))
	}

	if !opts.NoLog {
		s.ConfigureLogger()
	}

	// Run server in Go routine.
	go s.Start()

	if !s.ReadyForConnections(10 * time.Second) {
		panic(err)
	}
	return s
}

// LoadConfig loads a configuration from a filename
func LoadConfig(configFile string) (opts *server.Options) {
	opts, err := server.ProcessConfigFile(configFile)
	if err != nil {
		panic(fmt.Sprintf("Error processing configuration file: %v", err))
	}
	opts.NoSigs, opts.NoLog = true, opts.LogFile == ""
	return
}

func Up(configFile string) (s *server.Server, opts *server.Options) {
	opts = LoadConfig(configFile)
	s = RunServer(opts)
	return
}

func jsClientConnect(t testing.TB, s *server.Server, opts ...nats.Option) (*nats.Conn, nats.JetStreamContext) {
	t.Helper()
	nc, err := nats.Connect(s.ClientURL(), opts...)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	js, err := nc.JetStream(nats.MaxWait(10 * time.Second))
	if err != nil {
		t.Fatalf("Unexpected error getting JetStream context: %v", err)
	}
	return nc, js
}

func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func checkForErr(totalWait, sleepDur time.Duration, f func() error) error {
	timeout := time.Now().Add(totalWait)
	var err error
	for time.Now().Before(timeout) {
		err = f()
		if err == nil {
			return nil
		}
		time.Sleep(sleepDur)
	}
	return err
}

func CheckFor(t testing.TB, totalWait, sleepDur time.Duration, f func() error) {
	t.Helper()
	err := checkForErr(totalWait, sleepDur, f)
	if err != nil {
		t.Fatal(err.Error())
	}
}

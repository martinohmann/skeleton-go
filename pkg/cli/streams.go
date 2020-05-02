// Package cli contains utilities to work with IO streams and to improve CLI
// output, such as a custom log handler and a table writer.
package cli // import "kickoff.run/pkg/cli"

import (
	"bytes"
	"io"
	"os"
)

// DefaultIOStreams provides the default streams for os.Stdin, os.Stdout and
// os.Stderr.
var DefaultIOStreams = IOStreams{
	In:     os.Stdin,
	Out:    os.Stdout,
	ErrOut: os.Stderr,
}

// IOStreams is a holder for input and output streams. Commands should use this
// instead of directly relying on os.Stdin, os.Stdout and os.Stderr to make it
// possible to replace the streams in tests.
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

// NewTestIOStreams provides IOStreams that use a *bytes.Buffer. This can be
// used in tests to make assertions on command output as well as control the
// input stream. Returns IOStreams and *bytes.Buffer for in, out and errOut.
func NewTestIOStreams() (IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	in, out, errOut := &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}

	streams := IOStreams{
		In:     in,
		Out:    out,
		ErrOut: errOut,
	}

	return streams, in, out, errOut
}

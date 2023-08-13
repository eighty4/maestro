package composable

import (
	"fmt"
	"io"
	"os"
)

type Logging interface {
	Stdout() io.Writer
	Stderr() io.Writer
}

type WriterLogging struct {
	stdout io.Writer
	stderr io.Writer
}

func (l WriterLogging) Stdout() io.Writer {
	return l.stdout
}

func (l WriterLogging) Stderr() io.Writer {
	return l.stderr
}

type LabelledWriter struct {
	delegate io.Writer
	label    string
}

func (w LabelledWriter) Write(p []byte) (int, error) {
	_, err := w.delegate.Write(append([]byte(w.label), p...))
	return len(p), err
}

func ConsoleLogger(label string) Logging {
	if len(label) == 0 {
		return WriterLogging{
			stdout: os.Stdout,
			stderr: os.Stderr,
		}
	} else {
		label = fmt.Sprintf("[%s] ", label)
		return WriterLogging{
			stdout: LabelledWriter{
				delegate: os.Stdout,
				label:    label,
			},
			stderr: LabelledWriter{
				delegate: os.Stderr,
				label:    label,
			},
		}
	}
}

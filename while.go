package while

import (
	"context"
	"fmt"
	"io"

	yup "github.com/yupsh/framework"
	"github.com/yupsh/framework/opt"
	localopt "github.com/yupsh/while/opt"
)

// Flags represents the configuration options for the while command
type Flags = localopt.Flags

// LineProcessor is a function that processes a single line and returns a command
type LineProcessor func(line string) yup.Command

// CommandFunc is a helper type for creating commands from functions
type CommandFunc func(ctx context.Context, input io.Reader, output, stderr io.Writer) error

func (f CommandFunc) Execute(ctx context.Context, input io.Reader, output, stderr io.Writer) error {
	return f(ctx, input, output, stderr)
}

// Command implementation
type command struct {
	processor LineProcessor
	flags     Flags
}

// While creates a new while command that processes each line from input
// using the provided processor function
func While(processor LineProcessor, parameters ...any) yup.Command {
	args := opt.Args[string, Flags](parameters...)
	return command{
		processor: processor,
		flags:     args.Flags,
	}
}

func (c command) Execute(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) error {
	if c.processor == nil {
		return fmt.Errorf("while: processor function is required")
	}

	return yup.ProcessLinesSimple(ctx, stdin, stdout,
		func(ctx context.Context, lineNum int, line string, output io.Writer) error {
			// Apply the processor function to each line
			cmd := c.processor(line)
			if cmd == nil {
				return nil // Skip nil commands
			}

			// Execute the command for this line
			return cmd.Execute(ctx, nil, output, stderr)
		})
}

func (c command) String() string {
	return "while"
}

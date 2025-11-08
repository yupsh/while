package command

import (
	"bufio"
	"context"
	"io"
	"strings"

	gloo "github.com/gloo-foo/framework"
)

// Body is a function that processes input arguments and returns a Command to execute
type Body func(args ...any) gloo.Command

type command struct {
	body  Body
	flags flags
}

func While(body Body, parameters ...any) gloo.Command {
	inputs := gloo.Initialize[string, flags](parameters...)
	return command{
		body:  body,
		flags: inputs.Flags,
	}
}

func (c command) Executor() gloo.CommandExecutor {
	return func(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) error {
		// While loop that reads from stdin line by line
		// For each line, parse it according to FieldSeparator and call body function
		scanner := bufio.NewScanner(stdin)

		for scanner.Scan() {
			line := scanner.Text()

			// Parse line into fields based on FieldSeparator
			var args []any
			if c.flags.FieldSeparator != "" {
				// Split by field separator
				fields := strings.Split(line, string(c.flags.FieldSeparator))
				args = make([]any, len(fields))
				for i, field := range fields {
					args[i] = field
				}
			} else {
				// Default: split on whitespace
				fields := strings.Fields(line)
				args = make([]any, len(fields))
				for i, field := range fields {
					args[i] = field
				}
			}

			// Call body function with parsed arguments
			cmd := c.body(args...)
			if cmd == nil {
				// Body returned nil, skip this line
				continue
			}

			// Execute the command returned by body
			err := cmd.Executor()(ctx, strings.NewReader(""), stdout, stderr)
			if err != nil {
				return err
			}

			// Check for context cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}

		return scanner.Err()
	}
}

package command

import (
	"bufio"
	"context"
	"fmt"
	"io"

	yup "github.com/gloo-foo/framework"
)

type command yup.Inputs[string, flags]

func While(parameters ...any) yup.Command {
	return command(yup.Initialize[string, flags](parameters...))
}

func (p command) Executor() yup.CommandExecutor {
	return func(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) error {
		// While loop that reads from stdin and outputs until EOF
		// Simple pass-through implementation
		scanner := bufio.NewScanner(stdin)

		for scanner.Scan() {
			line := scanner.Text()
			_, err := fmt.Fprintln(stdout, line)
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

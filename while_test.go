package while

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	yup "github.com/yupsh/framework"
)

// Helper for creating simple commands in tests
type commandFunc func(ctx context.Context, input io.Reader, output, stderr io.Writer) error

func (f commandFunc) Execute(ctx context.Context, input io.Reader, output, stderr io.Writer) error {
	return f(ctx, input, output, stderr)
}

func TestWhileBasic(t *testing.T) {
	// Simple processor that prefixes each line with "processed: "
	processor := func(line string) yup.Command {
		return commandFunc(func(ctx context.Context, input io.Reader, output, stderr io.Writer) error {
			fmt.Fprintf(output, "processed: %s\n", line)
			return nil
		})
	}

	cmd := While(processor)

	input := "line1\nline2\nline3\n"
	expected := "processed: line1\nprocessed: line2\nprocessed: line3\n"

	var output strings.Builder
	var stderr strings.Builder

	ctx := context.Background()
	err := cmd.Execute(ctx, strings.NewReader(input), &output, &stderr)

	if err != nil {
		t.Fatalf("Execute failed: %v\nStderr: %s", err, stderr.String())
	}

	result := output.String()
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestWhileWithEcho(t *testing.T) {
	// Use a processor that creates echo commands
	processor := func(line string) yup.Command {
		return commandFunc(func(ctx context.Context, input io.Reader, output, stderr io.Writer) error {
			fmt.Fprintf(output, "echo: %s\n", strings.ToUpper(line))
			return nil
		})
	}

	cmd := While(processor)

	input := "hello\nworld\n"
	expected := "echo: HELLO\necho: WORLD\n"

	var output strings.Builder
	var stderr strings.Builder

	ctx := context.Background()
	err := cmd.Execute(ctx, strings.NewReader(input), &output, &stderr)

	if err != nil {
		t.Fatalf("Execute failed: %v\nStderr: %s", err, stderr.String())
	}

	result := output.String()
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestWhileWithNilCommand(t *testing.T) {
	// Processor that returns nil for empty lines
	processor := func(line string) yup.Command {
		if strings.TrimSpace(line) == "" {
			return nil // Skip empty lines
		}
		return commandFunc(func(ctx context.Context, input io.Reader, output, stderr io.Writer) error {
			fmt.Fprintf(output, "non-empty: %s\n", line)
			return nil
		})
	}

	cmd := While(processor)

	input := "line1\n\nline2\n   \nline3\n"
	expected := "non-empty: line1\nnon-empty: line2\nnon-empty: line3\n"

	var output strings.Builder
	var stderr strings.Builder

	ctx := context.Background()
	err := cmd.Execute(ctx, strings.NewReader(input), &output, &stderr)

	if err != nil {
		t.Fatalf("Execute failed: %v\nStderr: %s", err, stderr.String())
	}

	result := output.String()
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestWhileProcessorError(t *testing.T) {
	// Processor that returns an error for certain lines
	processor := func(line string) yup.Command {
		if strings.Contains(line, "error") {
			return commandFunc(func(ctx context.Context, input io.Reader, output, stderr io.Writer) error {
				return fmt.Errorf("simulated error for line: %s", line)
			})
		}
		return commandFunc(func(ctx context.Context, input io.Reader, output, stderr io.Writer) error {
			fmt.Fprintf(output, "ok: %s\n", line)
			return nil
		})
	}

	cmd := While(processor)

	input := "good\nerror line\nmore good\n"

	var output strings.Builder
	var stderr strings.Builder

	ctx := context.Background()
	err := cmd.Execute(ctx, strings.NewReader(input), &output, &stderr)

	// Should get an error
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "simulated error") {
		t.Errorf("Expected simulated error, got: %v", err)
	}
}

func TestWhileNoProcessor(t *testing.T) {
	// Test with nil processor
	cmd := While(nil)

	input := "test\n"

	var output strings.Builder
	var stderr strings.Builder

	ctx := context.Background()
	err := cmd.Execute(ctx, strings.NewReader(input), &output, &stderr)

	// Should get an error
	if err == nil {
		t.Error("Expected error for nil processor, got nil")
	}

	if !strings.Contains(err.Error(), "processor function is required") {
		t.Errorf("Expected processor required error, got: %v", err)
	}
}

func TestWhileContextCancellation(t *testing.T) {
	// Processor that does some work
	processor := func(line string) yup.Command {
		return commandFunc(func(ctx context.Context, input io.Reader, output, stderr io.Writer) error {
			// Check for cancellation
			if err := yup.CheckContextCancellation(ctx); err != nil {
				return err
			}
			fmt.Fprintf(output, "processed: %s\n", line)
			return nil
		})
	}

	cmd := While(processor)

	// Create a large input
	var inputBuilder strings.Builder
	for i := 0; i < 1000; i++ {
		inputBuilder.WriteString(fmt.Sprintf("line%d\n", i))
	}

	// Create a context that will be cancelled quickly
	ctx, cancel := context.WithCancel(context.Background())

	var output strings.Builder
	var stderr strings.Builder

	// Cancel context immediately
	cancel()

	err := cmd.Execute(ctx, strings.NewReader(inputBuilder.String()), &output, &stderr)

	// Should detect cancellation and return error
	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}

	if !strings.Contains(err.Error(), "context canceled") && !strings.Contains(err.Error(), "context cancelled") {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}
}

func TestWhileContextCancellationTimeout(t *testing.T) {
	processor := func(line string) yup.Command {
		return commandFunc(func(ctx context.Context, input io.Reader, output, stderr io.Writer) error {
			fmt.Fprintf(output, "processed: %s\n", line)
			return nil
		})
	}

	cmd := While(processor)

	// Create a large input
	var inputBuilder strings.Builder
	for i := 0; i < 1000; i++ {
		inputBuilder.WriteString(fmt.Sprintf("line%d\n", i))
	}

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	var output strings.Builder
	var stderr strings.Builder

	err := cmd.Execute(ctx, strings.NewReader(inputBuilder.String()), &output, &stderr)

	// Should timeout and return error (or succeed if fast enough)
	if err != nil {
		if !strings.Contains(err.Error(), "context") {
			t.Errorf("Expected context error, got: %v", err)
		}
	}
}

func TestWhileEmptyInput(t *testing.T) {
	processor := func(line string) yup.Command {
		return commandFunc(func(ctx context.Context, input io.Reader, output, stderr io.Writer) error {
			fmt.Fprintf(output, "processed: %s\n", line)
			return nil
		})
	}

	cmd := While(processor)

	input := ""
	expected := ""

	var output strings.Builder
	var stderr strings.Builder

	ctx := context.Background()
	err := cmd.Execute(ctx, strings.NewReader(input), &output, &stderr)

	if err != nil {
		t.Fatalf("Execute failed: %v\nStderr: %s", err, stderr.String())
	}

	result := output.String()
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestWhileString(t *testing.T) {
	processor := func(line string) yup.Command {
		return nil
	}

	cmd := While(processor)
	// Cast to our concrete type to access String method
	if whileCmd, ok := cmd.(command); ok {
		result := whileCmd.String()
		expected := "while"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	} else {
		t.Error("Command is not of expected type")
	}
}

func TestWhileInterface(t *testing.T) {
	// Verify that While command implements yup.Command interface
	processor := func(line string) yup.Command {
		return nil
	}
	var _ yup.Command = While(processor)
}

func BenchmarkWhileSimple(b *testing.B) {
	processor := func(line string) yup.Command {
		return commandFunc(func(ctx context.Context, input io.Reader, output, stderr io.Writer) error {
			fmt.Fprintf(output, "processed: %s\n", line)
			return nil
		})
	}

	cmd := While(processor)
	input := "line1\nline2\nline3\nline4\nline5\n"
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var output strings.Builder
		var stderr strings.Builder
		cmd.Execute(ctx, strings.NewReader(input), &output, &stderr)
	}
}

func BenchmarkWhileLarge(b *testing.B) {
	processor := func(line string) yup.Command {
		return commandFunc(func(ctx context.Context, input io.Reader, output, stderr io.Writer) error {
			fmt.Fprintf(output, "processed: %s\n", line)
			return nil
		})
	}

	cmd := While(processor)

	var inputBuilder strings.Builder
	for i := 0; i < 1000; i++ {
		inputBuilder.WriteString(fmt.Sprintf("line%d\n", i))
	}
	input := inputBuilder.String()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var output strings.Builder
		var stderr strings.Builder
		cmd.Execute(ctx, strings.NewReader(input), &output, &stderr)
	}
}

// Example tests for documentation
func ExampleWhile() {
	processor := func(line string) yup.Command {
		return commandFunc(func(ctx context.Context, input io.Reader, output, stderr io.Writer) error {
			fmt.Fprintf(output, ">>> %s\n", line)
			return nil
		})
	}

	cmd := While(processor)
	ctx := context.Background()

	input := strings.NewReader("hello\nworld\n")
	var output strings.Builder
	cmd.Execute(ctx, input, &output, &strings.Builder{})
	// Output would be: >>> hello\n>>> world\n
}

func ExampleWhile_withConditional() {
	processor := func(line string) yup.Command {
		if strings.HasPrefix(line, "#") {
			return nil // Skip comments
		}
		return commandFunc(func(ctx context.Context, input io.Reader, output, stderr io.Writer) error {
			fmt.Fprintf(output, "code: %s\n", line)
			return nil
		})
	}

	cmd := While(processor)
	ctx := context.Background()

	input := strings.NewReader("# comment\nactual code\n# another comment\nmore code\n")
	var output strings.Builder
	cmd.Execute(ctx, input, &output, &strings.Builder{})
	// Output would be: code: actual code\ncode: more code\n
}

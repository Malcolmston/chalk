package prompts

import (
	"io"
	"strconv"
	"strings"

	"github.com/malcolmston/chalk"
)

// InputConfig configures Input.
type InputConfig struct {
	// Message is the question shown to the user.
	Message string
	// Default is returned when the user submits an empty line.
	Default string
	// Validate, when set, must return nil to accept the input; a non-nil error
	// re-prompts with the error message.
	Validate func(string) error
	// Transform, when set, post-processes the accepted value before it is returned.
	Transform func(string) string
	// In is the input source (defaults to os.Stdin).
	In io.Reader
	// Out is the output destination (defaults to os.Stdout).
	Out io.Writer
}

// PasswordConfig configures Password.
type PasswordConfig struct {
	// Message is the question shown to the user.
	Message string
	// Validate, when set, must return nil to accept the input; a non-nil error
	// re-prompts with the error message.
	Validate func(string) error
	// Mask, when non-zero, echoes this rune for each character; when zero the
	// input is hidden entirely.
	Mask rune
	// In is the input source (defaults to os.Stdin).
	In io.Reader
	// Out is the output destination (defaults to os.Stdout).
	Out io.Writer
}

// ConfirmConfig configures Confirm.
type ConfirmConfig struct {
	// Message is the question shown to the user.
	Message string
	// Default is the answer used when the user submits an empty line.
	Default bool
	// In is the input source (defaults to os.Stdin).
	In io.Reader
	// Out is the output destination (defaults to os.Stdout).
	Out io.Writer
}

// NumberConfig configures Number.
type NumberConfig struct {
	// Message is the question shown to the user.
	Message string
	// Default, when non-nil, is returned for an empty line.
	Default *float64
	// Min, when non-nil, is the inclusive lower bound.
	Min *float64
	// Max, when non-nil, is the inclusive upper bound.
	Max *float64
	// Integer requires the value to be a whole number.
	Integer bool
	// Validate, when set, is an extra check applied to the parsed value.
	Validate func(float64) error
	// In is the input source (defaults to os.Stdin).
	In io.Reader
	// Out is the output destination (defaults to os.Stdout).
	Out io.Writer
}

// renderPrompt builds the "? message (default) " line.
func renderPrompt(message, def string) string {
	s := stylePrefix.Sprint("?") + " " + styleMessage.Sprint(message) + " "
	if def != "" {
		s += styleDim.Sprint("("+def+")") + " "
	}
	return s
}

// readLine reads a single line of input, echoing per the mask/hidden rules and
// re-prompting on validation failure.
func readLine(in io.Reader, out io.Writer, prompt string, mask rune, hidden bool, validate func(string) error) (string, error) {
	restore := enterRaw(in)
	defer restore()
	kr := newKeyReader(in)

	writeString(out, prompt)
	var buf []rune
	for {
		k := kr.read()
		switch k.typ {
		case keyEnter:
			line := string(buf)
			if validate != nil {
				if err := validate(line); err != nil {
					writeString(out, "\r\n"+styleError.Sprint("✖ "+err.Error())+"\r\n")
					writeString(out, prompt)
					buf = buf[:0]
					continue
				}
			}
			writeString(out, "\r\n")
			return line, nil
		case keyBackspace:
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				if !hidden {
					writeString(out, "\b \b")
				}
			}
		case keyCtrlC, keyEsc:
			writeString(out, "\r\n")
			return "", ErrCanceled
		case keyEOF:
			return string(buf), nil
		case keyRune, keySpace:
			buf = append(buf, k.r)
			switch {
			case hidden:
				// echo nothing
			case mask != 0:
				writeString(out, string(mask))
			default:
				writeString(out, string(k.r))
			}
		}
	}
}

// Input prompts for a line of text.
func Input(cfg InputConfig) (string, error) {
	in, out := resolveIO(cfg.In, cfg.Out)
	val, err := readLine(in, out, renderPrompt(cfg.Message, cfg.Default), 0, false, cfg.Validate)
	if err != nil {
		return "", err
	}
	if val == "" {
		val = cfg.Default
	}
	if cfg.Transform != nil {
		val = cfg.Transform(val)
	}
	return val, nil
}

// Password prompts for a hidden (or masked) secret.
func Password(cfg PasswordConfig) (string, error) {
	in, out := resolveIO(cfg.In, cfg.Out)
	hidden := cfg.Mask == 0
	return readLine(in, out, renderPrompt(cfg.Message, ""), cfg.Mask, hidden, cfg.Validate)
}

// Confirm prompts for a yes/no answer, returning the boolean result.
func Confirm(cfg ConfirmConfig) (bool, error) {
	in, out := resolveIO(cfg.In, cfg.Out)
	hint := "(y/N)"
	if cfg.Default {
		hint = "(Y/n)"
	}
	prompt := stylePrefix.Sprint("?") + " " + styleMessage.Sprint(cfg.Message) + " " + styleDim.Sprint(hint) + " "
	line, err := readLine(in, out, prompt, 0, false, nil)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return cfg.Default, nil
	}
}

// Number prompts for a numeric answer with optional bounds.
func Number(cfg NumberConfig) (float64, error) {
	in, out := resolveIO(cfg.In, cfg.Out)
	def := ""
	if cfg.Default != nil {
		def = strconv.FormatFloat(*cfg.Default, 'g', -1, 64)
	}

	validate := func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			if cfg.Default != nil {
				return nil
			}
			return errNumberRequired
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return errNotANumber
		}
		if cfg.Integer && v != float64(int64(v)) {
			return errNotAnInteger
		}
		if cfg.Min != nil && v < *cfg.Min {
			return &boundError{"must be >=", *cfg.Min}
		}
		if cfg.Max != nil && v > *cfg.Max {
			return &boundError{"must be <=", *cfg.Max}
		}
		if cfg.Validate != nil {
			return cfg.Validate(v)
		}
		return nil
	}

	line, err := readLine(in, out, renderPrompt(cfg.Message, def), 0, false, validate)
	if err != nil {
		return 0, err
	}
	line = strings.TrimSpace(line)
	if line == "" && cfg.Default != nil {
		return *cfg.Default, nil
	}
	return strconv.ParseFloat(line, 64)
}

type simpleError string

// Error implements the error interface.
func (e simpleError) Error() string { return string(e) }

const (
	errNumberRequired = simpleError("a number is required")
	errNotANumber     = simpleError("not a valid number")
	errNotAnInteger   = simpleError("must be a whole number")
)

type boundError struct {
	rel string
	n   float64
}

// Error implements the error interface.
func (e *boundError) Error() string {
	return e.rel + " " + chalk.Strip(strconv.FormatFloat(e.n, 'g', -1, 64))
}

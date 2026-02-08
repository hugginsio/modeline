// Copyright (c) Kyle Huggins
// SPDX-License-Identifier: BSD-3-Clause

package modeline

import (
	"errors"
	"strings"
)

// findProgram scans the line for the pattern: whitespace + word + colon.
// Returns the program identifier and the remaining text after the colon.
func findProgram(line string) (program, rest string, err error) {
	// Find whitespace followed by identifier and colon
	for i := 0; i < len(line); i++ {
		if line[i] == ' ' || line[i] == '\t' {
			// Found whitespace, look for identifier
			j := i + 1
			// Skip any additional whitespace
			for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
				j++
			}

			if j >= len(line) {
				continue
			}

			// Extract identifier (word characters)
			start := j
			for j < len(line) && isWordChar(line[j]) {
				j++
			}

			// Check if followed by colon
			if j < len(line) && line[j] == ':' && j > start {
				program = line[start:j]
				rest = line[j+1:]

				return program, rest, nil
			}
		}
	}

	return "", "", ErrNoModeline
}

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// parseOptions determines the form and extracts options from the remaining text.
func parseOptions(rest string) (map[string]string, error) {
	rest = strings.TrimLeft(rest, " \t")

	// Check for second form: starts with "set " or "se "
	if strings.HasPrefix(rest, "set ") {
		return parseSecondForm(rest[4:])
	}

	if strings.HasPrefix(rest, "se ") {
		return parseSecondForm(rest[3:])
	}

	// Check if it ends with : (malformed second form)
	if strings.HasSuffix(strings.TrimRight(rest, " \t"), ":") {
		return nil, errors.New("malformed modeline: ends with ':' but missing 'se[t]'")
	}

	// First form: split by whitespace and colons
	return parseFirstForm(rest), nil
}

// parseSecondForm extracts options from second form: options end at ':'.
func parseSecondForm(rest string) (map[string]string, error) {
	// Find the closing colon
	colonIdx := strings.Index(rest, ":")
	if colonIdx == -1 {
		// No closing colon, return empty options
		return make(map[string]string), nil
	}

	optionsText := rest[:colonIdx]
	return parseFirstForm(optionsText), nil
}

// parseFirstForm splits text by whitespace and colons, then parses each token.
func parseFirstForm(text string) map[string]string {
	options := make(map[string]string)

	// Split by whitespace and colons
	tokens := strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '\t' || r == ':'
	})

	for _, token := range tokens {
		if token == "" {
			continue
		}
		key, value := parseOption(token)
		if key != "" {
			options[key] = value
		}
	}

	return options
}

// parseOption parses a single option token into key and value.
func parseOption(token string) (key, value string) {
	// Check for key=value
	if key, val, found := strings.Cut(token, "="); found {
		return key, val
	}

	// Check for noXXX (boolean negation)
	if strings.HasPrefix(token, "no") && len(token) > 2 {
		return token[2:], "false"
	}

	// Plain token (implicit true)
	return token, "true"
}

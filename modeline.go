// Copyright (c) Kyle Huggins
// SPDX-License-Identifier: BSD-3-Clause

package modeline

import (
	"bufio"
	"errors"
	"io"
	"os"
)

var ErrNoModeline = errors.New("no modeline found")

// Modeline represents a parsed modeline.
type Modeline struct {
	Program string            // The identifier (e.g., "vi", "vim", "envctl")
	Options map[string]string // Parsed key=value options
	RawLine string            // Original line text
}

// Scanner extracts modelines from files or text.
type Scanner struct {
	ScanTop    bool // Scan from top of file.
	ScanBottom bool // Scan from bottom of file.
	MaxLines   int  // Lines to scan from each edge.
}

var defaultScanner = Scanner{
	ScanTop:    true,
	ScanBottom: true,
	MaxLines:   5,
}

// Scan extracts modelines from the reader.
func (s *Scanner) Scan(r io.Reader) ([]Modeline, error) {
	// Early return if neither top nor bottom scanning is enabled
	if !s.ScanTop && !s.ScanBottom {
		return []Modeline{}, nil
	}

	scanner := bufio.NewScanner(r)
	var modelines []Modeline

	// Optimize for top-only scanning: read and parse only MaxLines, then stop
	if s.ScanTop && !s.ScanBottom {
		lineCount := 0
		for scanner.Scan() && lineCount < s.MaxLines {
			line := scanner.Text()
			if m, err := s.ScanString(line); err == nil {
				modelines = append(modelines, *m)
			}

			lineCount++
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}

		return modelines, nil
	}

	// For bottom-only or both: use circular buffer for bottom lines
	bottomBuffer := make([]string, 0, s.MaxLines)
	lineCount := 0

	// If scanning top, parse the first MaxLines immediately
	if s.ScanTop {
		for scanner.Scan() && lineCount < s.MaxLines {
			line := scanner.Text()
			if m, err := s.ScanString(line); err == nil {
				modelines = append(modelines, *m)
			}
			lineCount++
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	// Continue reading remaining lines into circular buffer for bottom scanning
	for scanner.Scan() {
		line := scanner.Text()
		if len(bottomBuffer) < s.MaxLines {
			bottomBuffer = append(bottomBuffer, line)
		} else {
			// Circular buffer: shift and add new line
			copy(bottomBuffer, bottomBuffer[1:])
			bottomBuffer[s.MaxLines-1] = line
		}
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Parse bottom lines from buffer
	if s.ScanBottom {
		// Determine which lines to scan from buffer to avoid duplicates
		startIdx := 0
		if s.ScanTop && lineCount <= s.MaxLines {
			// File is shorter than or equal to MaxLines, and we already scanned from top
			// Don't scan any lines from bottom buffer (they were already scanned)
			return modelines, nil
		} else if s.ScanTop && lineCount < 2*s.MaxLines {
			// File is shorter than 2*MaxLines
			// Skip the overlap: we already scanned first MaxLines
			overlap := 2*s.MaxLines - lineCount
			startIdx = s.MaxLines - overlap
		}

		for i := startIdx; i < len(bottomBuffer); i++ {
			if m, err := s.ScanString(bottomBuffer[i]); err == nil {
				modelines = append(modelines, *m)
			}
		}
	}

	return modelines, nil
}

// ScanFile extracts modelines from a file.
func (s *Scanner) ScanFile(path string) ([]Modeline, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()
	return s.Scan(file)
}

// ScanString is a convenience method for extracting a modeline from a single string.
func (s *Scanner) ScanString(str string) (*Modeline, error) {
	program, rest, err := findProgram(str)
	if err != nil {
		return nil, err
	}

	options, err := parseOptions(rest)
	if err != nil {
		return nil, err
	}

	return &Modeline{
		Program: program,
		Options: options,
		RawLine: str,
	}, nil
}

// Scan extracts modelines from the reader using the default settings.
func Scan(r io.Reader) ([]Modeline, error) {
	return defaultScanner.Scan(r)
}

// ScanFile extracts modelines from a file using the default settings.
func ScanFile(path string) ([]Modeline, error) {
	return defaultScanner.ScanFile(path)
}

// ScanString is a convenience method for extracting a modeline from a single string.
func ScanString(str string) (*Modeline, error) {
	return defaultScanner.ScanString(str)
}

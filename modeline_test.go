// Copyright (c) Kyle Huggins
// SPDX-License-Identifier: BSD-3-Clause

package modeline_test

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/hugginsio/modeline"
)

//go:embed modeline_test.txt
var testFileContent string

// compareModeline checks if two Modeline structs are equal.
func compareModeline(t *testing.T, got, want *modeline.Modeline) bool {
	t.Helper()
	if got.Program != want.Program {
		t.Errorf("Program = %q, want %q", got.Program, want.Program)
		return false
	}

	if len(got.Options) != len(want.Options) {
		t.Errorf("got %d options, want %d", len(got.Options), len(want.Options))
		return false
	}

	for k, v := range want.Options {
		if gotVal, ok := got.Options[k]; !ok {
			t.Errorf("missing option %q", k)
			return false
		} else if gotVal != v {
			t.Errorf("Options[%q] = %q, want %q", k, gotVal, v)
			return false
		}
	}

	if got.RawLine != want.RawLine {
		t.Errorf("RawLine = %q, want %q", got.RawLine, want.RawLine)
		return false
	}

	return true
}

func TestScanString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *modeline.Modeline
		wantErr bool
	}{
		{
			name:  "envctl",
			input: "# envctl: provider=gsm gsm_project=526782592",
			want: &modeline.Modeline{
				Program: "envctl",
				Options: map[string]string{
					"provider":    "gsm",
					"gsm_project": "526782592",
				},
			},
		},
		{
			name:  "first form key=value",
			input: "# vim:sw=3:foldmethod=marker",
			want: &modeline.Modeline{
				Program: "vim",
				Options: map[string]string{
					"sw":         "3",
					"foldmethod": "marker",
				},
			},
		},
		{
			name:  "first form key=value with whitespace",
			input: "# vim: sw=3 foldmethod=marker",
			want: &modeline.Modeline{
				Program: "vim",
				Options: map[string]string{
					"sw":         "3",
					"foldmethod": "marker",
				},
			},
		},
		{
			name:  "first form implicit boolean",
			input: "# vim:noai:cursorline",
			want: &modeline.Modeline{
				Program: "vim",
				Options: map[string]string{
					"ai":         "false",
					"cursorline": "true",
				},
			},
		},
		{
			name:  "first form implicit boolean with whitespace",
			input: "# vim: noai cursorline",
			want: &modeline.Modeline{
				Program: "vim",
				Options: map[string]string{
					"ai":         "false",
					"cursorline": "true",
				},
			},
		},
		{
			name:  "second form key=value",
			input: "/* vim:set sw=3 foldmethod=marker: */ random text",
			want: &modeline.Modeline{
				Program: "vim",
				Options: map[string]string{
					"sw":         "3",
					"foldmethod": "marker",
				},
			},
		},
		{
			name:  "second form key=value with whitespace",
			input: "# vim: set sw=3 foldmethod=marker: other text",
			want: &modeline.Modeline{
				Program: "vim",
				Options: map[string]string{
					"sw":         "3",
					"foldmethod": "marker",
				},
			},
		},
		{
			name:  "second form implicit boolean",
			input: "# vim:se noai cursorline: boolin'",
			want: &modeline.Modeline{
				Program: "vim",
				Options: map[string]string{
					"ai":         "false",
					"cursorline": "true",
				},
			},
		},
		{
			name:  "second form implicit boolean with whitespace",
			input: "/*** vim: se noai cursorline: ***/",
			want: &modeline.Modeline{
				Program: "vim",
				Options: map[string]string{
					"ai":         "false",
					"cursorline": "true",
				},
			},
		},
		{
			name:    "lack of leading text",
			input:   "robot: beep=boop",
			wantErr: true,
		},
		{
			name:  "second form missing closing colon",
			input: "# robot: se beep=boop",
			want: &modeline.Modeline{
				Program: "robot",
				Options: map[string]string{},
			},
		},
		{
			name:    "second form missing se[t]",
			input:   "# robot: beep=boop:",
			wantErr: true,
		},
		{
			name:    "not a modeline",
			input:   "# not a modeline",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := modeline.ScanString(tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("ScanString(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Override input since I don't repeat RawLine in the spec
			tt.want.RawLine = tt.input

			compareModeline(t, got, tt.want)
		})
	}
}

func TestScan(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		scanner  modeline.Scanner
		wantLen  int
		wantMods []modeline.Modeline
	}{
		{
			name:    "empty file",
			input:   "",
			scanner: modeline.Scanner{ScanTop: true, ScanBottom: true, MaxLines: 5},
			wantLen: 0,
		},
		{
			name: "no modelines",
			input: `line 1
line 2
line 3
`,
			scanner: modeline.Scanner{ScanTop: true, ScanBottom: true, MaxLines: 5},
			wantLen: 0,
		},
		{
			name: "scan top only, modeline in first line",
			input: `# vim: sw=3
line 2
line 3
`,
			scanner: modeline.Scanner{ScanTop: true, ScanBottom: false, MaxLines: 5},
			wantLen: 1,
			wantMods: []modeline.Modeline{
				{
					Program: "vim",
					Options: map[string]string{"sw": "3"},
					RawLine: "# vim: sw=3",
				},
			},
		},
		{
			name: "scan top only, modeline within MaxLines",
			input: `line 1
line 2
# vim: sw=3
line 4
line 5
line 6
`,
			scanner: modeline.Scanner{ScanTop: true, ScanBottom: false, MaxLines: 5},
			wantLen: 1,
			wantMods: []modeline.Modeline{
				{
					Program: "vim",
					Options: map[string]string{"sw": "3"},
					RawLine: "# vim: sw=3",
				},
			},
		},
		{
			name: "scan top only, modeline beyond MaxLines",
			input: `line 1
line 2
line 3
line 4
line 5
line 6
# vim: sw=3
`,
			scanner: modeline.Scanner{ScanTop: true, ScanBottom: false, MaxLines: 5},
			wantLen: 0,
		},
		{
			name: "scan bottom only, modeline in last line",
			input: `line 1
line 2
# vim: sw=3
`,
			scanner: modeline.Scanner{ScanTop: false, ScanBottom: true, MaxLines: 5},
			wantLen: 1,
			wantMods: []modeline.Modeline{
				{
					Program: "vim",
					Options: map[string]string{"sw": "3"},
					RawLine: "# vim: sw=3",
				},
			},
		},
		{
			name: "scan bottom only, modeline within MaxLines from bottom",
			input: `line 1
line 2
# vim: sw=3
line 4
line 5
line 6
`,
			scanner: modeline.Scanner{ScanTop: false, ScanBottom: true, MaxLines: 5},
			wantLen: 1,
			wantMods: []modeline.Modeline{
				{
					Program: "vim",
					Options: map[string]string{"sw": "3"},
					RawLine: "# vim: sw=3",
				},
			},
		},
		{
			name: "scan bottom only, modeline beyond MaxLines from bottom",
			input: `# vim: sw=3
line 2
line 3
line 4
line 5
line 6
`,
			scanner: modeline.Scanner{ScanTop: false, ScanBottom: true, MaxLines: 5},
			wantLen: 0,
		},
		{
			name: "scan both, modelines in top and bottom",
			input: `# vim: sw=3
line 2
line 3
line 4
line 5
line 6
# envctl: provider=gsm
`,
			scanner: modeline.Scanner{ScanTop: true, ScanBottom: true, MaxLines: 2},
			wantLen: 2,
			wantMods: []modeline.Modeline{
				{
					Program: "vim",
					Options: map[string]string{"sw": "3"},
					RawLine: "# vim: sw=3",
				},
				{
					Program: "envctl",
					Options: map[string]string{"provider": "gsm"},
					RawLine: "# envctl: provider=gsm",
				},
			},
		},
		{
			name: "multiple modelines in top section",
			input: `# vim: sw=3
# envctl: provider=gsm
line 3
line 4
`,
			scanner: modeline.Scanner{ScanTop: true, ScanBottom: false, MaxLines: 5},
			wantLen: 2,
			wantMods: []modeline.Modeline{
				{
					Program: "vim",
					Options: map[string]string{"sw": "3"},
					RawLine: "# vim: sw=3",
				},
				{
					Program: "envctl",
					Options: map[string]string{"provider": "gsm"},
					RawLine: "# envctl: provider=gsm",
				},
			},
		},
		{
			name: "mixed valid and invalid lines",
			input: `# vim: sw=3
invalid line
# envctl: provider=gsm
`,
			scanner: modeline.Scanner{ScanTop: true, ScanBottom: false, MaxLines: 5},
			wantLen: 2,
			wantMods: []modeline.Modeline{
				{
					Program: "vim",
					Options: map[string]string{"sw": "3"},
					RawLine: "# vim: sw=3",
				},
				{
					Program: "envctl",
					Options: map[string]string{"provider": "gsm"},
					RawLine: "# envctl: provider=gsm",
				},
			},
		},
		{
			name: "file shorter than MaxLines",
			input: `# vim: sw=3
line 2
`,
			scanner: modeline.Scanner{ScanTop: true, ScanBottom: true, MaxLines: 10},
			wantLen: 1,
			wantMods: []modeline.Modeline{
				{
					Program: "vim",
					Options: map[string]string{"sw": "3"},
					RawLine: "# vim: sw=3",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			got, err := tt.scanner.Scan(reader)

			if err != nil {
				t.Fatalf("Scan() error = %v", err)
			}

			if len(got) != tt.wantLen {
				t.Fatalf("Scan() got %d modelines, want %d", len(got), tt.wantLen)
			}

			for i, wantMod := range tt.wantMods {
				compareModeline(t, &got[i], &wantMod)
			}
		})
	}
}

func TestScanFile(t *testing.T) {
	// modeline_test.txt has modelines at the top (lines 1-5) and bottom (lines 96-100)
	// Default scanner has ScanTop=true, ScanBottom=true, MaxLines=5
	got, err := modeline.ScanFile("modeline_test.txt")
	if err != nil {
		t.Fatalf("ScanFile() error = %v", err)
	}

	// Expect 10 modelines total (5 from top, 5 from bottom)
	if len(got) != 10 {
		t.Fatalf("ScanFile() got %d modelines, want 10", len(got))
	}

	// Verify the first modeline from top
	if got[0].Program != "vim" {
		t.Errorf("First modeline program = %q, want %q", got[0].Program, "vim")
	}

	if got[0].RawLine != "# vim:sw=3:foldmethod=marker" {
		t.Errorf("First modeline RawLine = %q, want %q", got[0].RawLine, "# vim:sw=3:foldmethod=marker")
	}

	// Verify the last modeline from bottom
	lastIdx := len(got) - 1
	if got[lastIdx].Program != "vim" {
		t.Errorf("Last modeline program = %q, want %q", got[lastIdx].Program, "vim")
	}
	if got[lastIdx].RawLine != "# vim:se noai cursorline: boolin'" {
		t.Errorf("Last modeline RawLine = %q, want %q", got[lastIdx].RawLine, "# vim:se noai cursorline: boolin'")
	}
}

func BenchmarkScan(b *testing.B) {
	scanner := modeline.Scanner{
		ScanTop:    true,
		ScanBottom: true,
		MaxLines:   5,
	}

	b.ReportAllocs()

	for b.Loop() {
		reader := strings.NewReader(testFileContent)
		modelines, err := scanner.Scan(reader)

		if err != nil {
			b.Fatalf("Scan() error = %v", err)
		}

		if len(modelines) != 10 {
			b.Fatalf("Expected 10 modelines, got %v", len(modelines))
		}
	}
}

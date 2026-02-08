# modeline

A flexible Vim-style modeline parser.

There are two forms of modelines, according to `:help modeline`. This module endeavours to support both.

1. `[text]{white}{program:}[white]{options}`
2. `[text]{white}{program:}[white]se[t] {options}:[text]`

A modeline such as `# vim: sw=3 foldmethod=marker noai cursorline` would result in the following:

```go
&modeline.Modeline{
	Program: "vim",
	Options: map[string]string{
		"sw":         "3",
		"foldmethod": "marker",
		"ai":         "false",
		"cursorline": "true",
	},
}
```

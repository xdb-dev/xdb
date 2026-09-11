module github.com/xdb-dev/xdb/internal/evals

go 1.26.0

require (
	github.com/openai/openai-go/v3 v3.49.0
	github.com/sonnes/pi-go v0.0.2-0.20260909063218-0b4d7a999dc8
	github.com/sonnes/pi-go/pkg/ai/provider/openairesponses v0.1.0
	github.com/stretchr/testify v1.11.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/charmbracelet/colorprofile v0.2.3-0.20250311203215-f60798e515dc // indirect
	github.com/charmbracelet/lipgloss v1.1.0 // indirect
	github.com/charmbracelet/log v1.0.0 // indirect
	github.com/charmbracelet/x/ansi v0.8.0 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.13-0.20250311204145-2c3ea96c31dd // indirect
	github.com/charmbracelet/x/term v0.2.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logfmt/logfmt v0.6.1 // indirect
	github.com/google/jsonschema-go v0.4.2 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sonnes/pi-go/pkg/ai/provider/openai v0.0.0 // indirect
	github.com/tidwall/gjson v1.19.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d // indirect
	golang.org/x/sys v0.47.0 // indirect
)

// pi-go's provider modules require github.com/sonnes/pi-go/pkg/ai/provider/openai
// v0.0.0, a version that has no tag. Their own go.mod resolves it with a
// replace, and Go ignores a replace in a dependency. So the replace has to be
// here. Remove these once pi-go tags its provider modules against real
// versions.
replace (
	github.com/sonnes/pi-go => ../../../pi-go
	github.com/sonnes/pi-go/pkg/ai/provider/openai => ../../../pi-go/pkg/ai/provider/openai
	github.com/sonnes/pi-go/pkg/ai/provider/openairesponses => ../../../pi-go/pkg/ai/provider/openairesponses
)

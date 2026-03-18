package cli

// commandHelpTemplate is used for leaf commands (methods like records create).
// It omits the GLOBAL OPTIONS section to reduce noise.
var commandHelpTemplate = `NAME:
   {{template "helpNameTemplate" .}}

USAGE:
   {{if .UsageText}}{{wrap .UsageText 3}}{{else}}{{.FullName}}{{if .VisibleFlags}} [options]{{end}}{{if .ArgsUsage}} {{.ArgsUsage}}{{end}}{{end}}

{{- if .Description}}

DESCRIPTION:
   {{template "descriptionTemplate" .}}{{end}}

{{- if .VisibleCommands}}

COMMANDS:{{template "visibleCommandTemplate" .}}{{end}}

{{- if .VisibleFlagCategories}}

OPTIONS:{{template "visibleFlagCategoryTemplate" .}}{{else if .VisibleFlags}}

OPTIONS:{{template "visibleFlagTemplate" .}}{{end}}
`

// subcommandHelpTemplate is used for resource-level commands (records, schemas, etc.).
// It omits the CATEGORY line.
var subcommandHelpTemplate = `NAME:
   {{template "helpNameTemplate" .}}

USAGE:
   {{if .UsageText}}{{wrap .UsageText 3}}{{else}}{{.FullName}} <method> [options]{{end}}
{{- if .Description}}

DESCRIPTION:
   {{template "descriptionTemplate" .}}{{end}}
{{- if .VisibleCommands}}

METHODS:{{template "visibleCommandTemplate" .}}{{end}}
{{- if .VisibleFlagCategories}}

OPTIONS:{{template "visibleFlagCategoryTemplate" .}}{{else if .VisibleFlags}}

OPTIONS:{{template "visibleFlagTemplate" .}}{{end}}
`

var rootHelpTemplate = `xdb — Tuple-based data store

USAGE:
    xdb <resource> <method> [flags]
    xdb <alias> <uri> [flags]
    xdb describe <resource.method | TypeName>

EXAMPLES:
    xdb records create --uri xdb://com.example/posts/post-1 --json '{"title":"Hello"}'
    xdb records list   --uri xdb://com.example/posts --fields id,title --limit 10
    xdb get xdb://com.example/posts/post-1
    xdb describe records.create

FLAGS:
    --config, -c <PATH>   Path to config file (default: ~/.xdb/config.json)
    --output, -o <FMT>    Output format: json (default), table, yaml, ndjson
    --verbose, -v         Enable verbose logging
    --debug               Enable debug logging

RESOURCES:{{range .VisibleCategories}}{{if eq .Name "resources"}}{{range .VisibleCommands}}
    {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}

OPERATIONS:{{range .VisibleCategories}}{{if eq .Name "operations"}}{{range .VisibleCommands}}
    {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}

ALIASES:{{range .VisibleCategories}}{{if eq .Name "aliases"}}{{range .VisibleCommands}}
    {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}

AGENT:{{range .VisibleCategories}}{{if eq .Name "agent"}}{{range .VisibleCommands}}
    {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}

SYSTEM:{{range .VisibleCategories}}{{if eq .Name "system"}}{{range .VisibleCommands}}
    {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}

EXIT CODES:
    0    Success
    1    Application error (NOT_FOUND, ALREADY_EXISTS, SCHEMA_VIOLATION)
    2    Connection error (daemon not running, timeout)
    3    Input validation error (INVALID_URI, bad JSON)
    4    Internal error (store failure, unexpected daemon error)
`

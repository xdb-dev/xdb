package cli

import (
	"context"
	"embed"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/urfave/cli/v3"
)

//go:embed skills/*/SKILL.md
var skillFiles embed.FS

// skill holds a parsed skill document.
type skill struct {
	Name        string
	Description string
	Category    string
	Content     string
}

// skills is populated from embedded SKILL.md files at init.
var skills map[string]skill

func init() {
	skills = make(map[string]skill)

	entries, err := skillFiles.ReadDir("skills")
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		path := "skills/" + name + "/SKILL.md"

		data, readErr := skillFiles.ReadFile(path)
		if readErr != nil {
			continue
		}

		s := parseSkillFrontmatter(name, string(data))
		skills[s.Name] = s
	}
}

// parseSkillFrontmatter extracts YAML frontmatter and content from a SKILL.md.
func parseSkillFrontmatter(dirName, raw string) skill {
	s := skill{Name: dirName, Category: "recipe"}

	// Split on --- delimiters.
	if !strings.HasPrefix(raw, "---\n") {
		s.Content = raw
		return s
	}

	rest := raw[4:] // skip opening ---\n
	end := strings.Index(rest, "\n---\n")

	if end < 0 {
		s.Content = raw
		return s
	}

	frontmatter := rest[:end]
	s.Content = strings.TrimLeft(rest[end+4:], "\n")

	// Simple line-by-line YAML parsing (no dependency needed).
	for _, line := range strings.Split(frontmatter, "\n") {
		key, val, found := strings.Cut(line, ":")
		if !found {
			continue
		}

		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"`)

		switch strings.TrimSpace(key) {
		case "name":
			s.Name = val
		case "description":
			s.Description = val
		case "category":
			s.Category = val
		}
	}

	return s
}

func skillsCmd() *cli.Command {
	return &cli.Command{
		Name:               "skills",
		Usage:              "Discover and fetch skill documents",
		Category:           "agent",
		CustomHelpTemplate: subcommandHelpTemplate,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
		},
		Commands: []*cli.Command{
			{
				Name:               "get",
				Usage:              "Fetch a specific skill",
				CustomHelpTemplate: commandHelpTemplate,
				ArgsUsage:          "<skill-name>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format"},
				},
				Action: skillsGetAction,
			},
		},
		Action: skillsListAction,
	}
}

func skillsListAction(_ context.Context, cmd *cli.Command) error {
	names := make([]string, 0, len(skills))
	for name := range skills {
		names = append(names, name)
	}

	sort.Strings(names)

	items := make([]any, len(names))
	for i, name := range names {
		s := skills[name]
		items[i] = map[string]string{
			"name":        s.Name,
			"description": s.Description,
			"category":    s.Category,
		}
	}

	return formatList(cmd, items)
}

func skillsGetAction(_ context.Context, cmd *cli.Command) error {
	args := cmd.Args()
	if args.Len() == 0 {
		return fmt.Errorf("skill name required")
	}

	name := args.First()

	s, ok := skills[name]
	if !ok {
		return fmt.Errorf("unknown skill: %s\nrun 'xdb skills' to list available skills", name)
	}

	_, err := fmt.Fprint(os.Stdout, s.Content)

	return err
}

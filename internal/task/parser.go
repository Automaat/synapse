package task

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

var frontmatterRe = regexp.MustCompile(`(?m)^---\s*$`)

func Parse(path string) (Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Task{}, fmt.Errorf("read task file: %w", err)
	}

	t, err := ParseBytes(data)
	if err != nil {
		return Task{}, fmt.Errorf("parse %s: %w", path, err)
	}
	t.FilePath = path
	return t, nil
}

func ParseBytes(data []byte) (Task, error) {
	locs := frontmatterRe.FindAllIndex(data, 2)
	if len(locs) < 2 {
		return Task{}, fmt.Errorf("invalid frontmatter: expected --- delimiters")
	}

	fm := data[locs[0][1]:locs[1][0]]

	var t Task
	if err := yaml.Unmarshal(fm, &t); err != nil {
		return Task{}, fmt.Errorf("unmarshal frontmatter: %w", err)
	}

	t.Body = string(bytes.TrimSpace(data[locs[1][1]:]))
	return t, nil
}

func Marshal(t Task) ([]byte, error) {
	t.UpdatedAt = time.Now().UTC()

	fm, err := yaml.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("marshal frontmatter: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("---" + "\n")
	buf.Write(fm)
	buf.WriteString("---" + "\n")
	if t.Body != "" {
		buf.WriteString(t.Body)
		buf.WriteString("\n")
	}
	return buf.Bytes(), nil
}

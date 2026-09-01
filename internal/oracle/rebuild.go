// Package oracle is the independent cold rebuild path. Cache verification never
// supplies input to this package; it rebuilds the semantic program and Go
// artifact from the current .gooo source on every conformance case.
package oracle

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/kimjooyoon/gooo-proof-carrying-semantic-cache/internal/protocol"
)

func Rebuild(sourcePath string, raw []byte, policy protocol.Policy) (protocol.SemanticIR, protocol.GeneratedArtifact, error) {
	program, canonical, err := parseSource(string(raw), policy)
	if err != nil {
		return protocol.SemanticIR{}, protocol.GeneratedArtifact{}, fmt.Errorf("rebuild %s: %w", sourcePath, err)
	}
	ir := protocol.SemanticIR{
		Schema:      "gooo/proof-carrying-semantic-cache/semantic-ir/v1",
		SourcePath:  sourcePath,
		RawDigest:   protocol.Digest(raw),
		SemanticKey: protocol.Digest([]byte(canonical)),
		Canonical:   canonical,
		Program:     program,
		Terminal:    protocol.TerminalWitness{Reason: program.Reason, Effect: program.Effect},
	}
	artifactBytes, err := render(program, policy)
	if err != nil {
		return protocol.SemanticIR{}, protocol.GeneratedArtifact{}, err
	}
	artifact := protocol.GeneratedArtifact{
		Path:   "generated/main.go",
		Bytes:  artifactBytes,
		Digest: protocol.Digest(artifactBytes),
	}
	return ir, artifact, nil
}

func parseSource(text string, policy protocol.Policy) (protocol.Program, string, error) {
	var program protocol.Program
	var canonicalLines []string
	scanner := bufio.NewScanner(strings.NewReader(text))
	for lineNo := 1; scanner.Scan(); lineNo++ {
		line := stripComment(scanner.Text())
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		tokens, err := sourceTokens(line)
		if err != nil {
			return protocol.Program{}, "", fmt.Errorf("line %d: %w", lineNo, err)
		}
		switch tokens[0] {
		case "program":
			if len(tokens) != 2 || program.Name != "" {
				return protocol.Program{}, "", fmt.Errorf("line %d: malformed program declaration", lineNo)
			}
			program.Name = tokens[1]
			canonicalLines = append(canonicalLines, "program "+quote(tokens[1]))
		case "terminal":
			if len(tokens) != 5 || tokens[1] != "reason" || tokens[3] != "effect" || program.Reason != "" {
				return protocol.Program{}, "", fmt.Errorf("line %d: malformed terminal declaration", lineNo)
			}
			program.Reason = tokens[2]
			program.Effect = tokens[4]
			canonicalLines = append(canonicalLines, "terminal reason "+quote(tokens[2])+" effect "+quote(tokens[4]))
		default:
			return protocol.Program{}, "", fmt.Errorf("line %d: unknown source record %q", lineNo, tokens[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return protocol.Program{}, "", err
	}
	if program.Name == "" || program.Reason == "" || program.Effect == "" {
		return protocol.Program{}, "", fmt.Errorf("source must declare program and terminal witness")
	}
	if policy.Normalization["comments"] != "ignored" || policy.Normalization["whitespace"] != "collapsed" || policy.Normalization["string_literals"] != "preserved" {
		return protocol.Program{}, "", fmt.Errorf("source normalization is not declared by contract")
	}
	return program, strings.Join(canonicalLines, "\n") + "\n", nil
}

func render(program protocol.Program, policy protocol.Policy) ([]byte, error) {
	if policy.Generation.Language != "go" || policy.Generation.Package != "main" || policy.Generation.Entrypoint != "main" {
		return nil, fmt.Errorf("unsupported generation plan")
	}
	message, err := json.Marshal("program=" + program.Name + " reason=" + program.Reason + " effect=" + program.Effect)
	if err != nil {
		return nil, err
	}
	text := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(" + string(message) + ")\n}\n"
	return []byte(text), nil
}

func sourceTokens(line string) ([]string, error) {
	var tokens []string
	for i := 0; i < len(line); {
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
		if i == len(line) {
			break
		}
		if line[i] != '"' {
			start := i
			for i < len(line) && line[i] != ' ' && line[i] != '\t' {
				i++
			}
			tokens = append(tokens, line[start:i])
			continue
		}
		start := i
		i++
		escaped := false
		closed := false
		for i < len(line) {
			if escaped {
				escaped = false
				i++
				continue
			}
			if line[i] == '\\' {
				escaped = true
				i++
				continue
			}
			if line[i] == '"' {
				i++
				closed = true
				break
			}
			i++
		}
		if !closed {
			return nil, fmt.Errorf("unterminated quoted value")
		}
		value, err := strconv.Unquote(line[start:i])
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, value)
	}
	return tokens, nil
}

func stripComment(line string) string {
	if index := strings.Index(line, "//"); index >= 0 {
		return line[:index]
	}
	return line
}

func quote(value string) string {
	return strconv.Quote(value)
}

package protocol

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func LoadPolicy(path string) (Policy, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, "", err
	}
	policy, err := ParsePolicy(string(raw))
	if err != nil {
		return Policy{}, "", fmt.Errorf("%s: %w", path, err)
	}
	return policy, Digest(raw), nil
}

func ParsePolicy(text string) (Policy, error) {
	var policy Policy
	policy.Normalization = map[string]string{}
	policy.CacheKey = map[string]string{}
	policy.Fallback = map[Decision]string{}
	scanner := bufio.NewScanner(strings.NewReader(text))
	lineNo := 0
	inside := false
	seenHeader := false
	for scanner.Scan() {
		lineNo++
		line := stripMetaComment(scanner.Text())
		if strings.TrimSpace(line) == "" {
			continue
		}
		tokens, err := tokenize(line)
		if err != nil {
			return Policy{}, fmt.Errorf("line %d: %w", lineNo, err)
		}
		if !seenHeader {
			if len(tokens) != 3 || tokens[0] != "contract" || tokens[2] != "{" {
				return Policy{}, fmt.Errorf("line %d: expected contract header", lineNo)
			}
			policy.Schema = tokens[1]
			seenHeader, inside = true, true
			continue
		}
		if len(tokens) == 1 && tokens[0] == "}" {
			if !inside {
				return Policy{}, fmt.Errorf("line %d: unexpected closing brace", lineNo)
			}
			inside = false
			continue
		}
		if !inside {
			return Policy{}, fmt.Errorf("line %d: content after contract body", lineNo)
		}
		if len(tokens) == 0 {
			continue
		}
		switch tokens[0] {
		case "authority":
			if len(tokens) != 2 {
				return Policy{}, fmt.Errorf("line %d: authority requires one value", lineNo)
			}
			policy.Authority = tokens[1]
		case "precedence":
			if len(tokens) < 2 {
				return Policy{}, fmt.Errorf("line %d: precedence is empty", lineNo)
			}
			for _, token := range tokens[1:] {
				policy.Precedence = append(policy.Precedence, Decision(token))
			}
		case "unknown_fields":
			policy.UnknownFields = append([]string(nil), tokens[1:]...)
		case "denominator":
			pairs, err := pairsAfter(tokens, 2)
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			policy.Denominator.ID = tokens[1]
			policy.Denominator.Count, err = strconv.Atoi(pairs["count"])
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: malformed denominator count", lineNo)
			}
		case "normalization":
			pairs, err := pairsAfter(tokens, 1)
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			for key, value := range pairs {
				policy.Normalization[key] = value
			}
		case "cache_key":
			pairs, err := pairsAfter(tokens, 1)
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			for key, value := range pairs {
				policy.CacheKey[key] = value
			}
		case "binding":
			if len(tokens) < 3 {
				return Policy{}, fmt.Errorf("line %d: binding kind is required", lineNo)
			}
			pairs, err := pairsAfter(tokens, 2)
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			policy.Bindings = append(policy.Bindings, BindingRule{
				Kind: tokens[1], Stage: pairs["stage"], Step: pairs["step"],
				Mismatch: Decision(pairs["mismatch"]), UnknownClass: pairs["unknown_class"],
			})
		case "obligation":
			if len(tokens) < 3 {
				return Policy{}, fmt.Errorf("line %d: obligation id is required", lineNo)
			}
			pairs, err := pairsAfter(tokens, 2)
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			policy.Obligations = append(policy.Obligations, ProofObligation{
				ID: tokens[1], Stage: pairs["stage"], Step: pairs["step"],
				Proof: pairs["proof"], Missing: pairs["missing"],
			})
		case "witness":
			if len(tokens) < 3 {
				return Policy{}, fmt.Errorf("line %d: witness kind is required", lineNo)
			}
			pairs, err := pairsAfter(tokens, 2)
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			policy.Witness = WitnessRule{Kind: tokens[1], Stage: pairs["stage"], Step: pairs["step"], Reason: pairs["reason"], Effect: pairs["effect"], Mismatch: Decision(pairs["mismatch"])}
		case "reuse":
			pairs, err := pairsAfter(tokens, 1)
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			policy.Reuse = PolicyRule{Name: pairs["rule"], Status: Decision(pairs["status"]), Requires: pairs["requires"]}
		case "fallback":
			pairs, err := pairsAfter(tokens, 1)
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			if pairs["status"] == "" || pairs["operation"] == "" || pairs["unknown_class"] == "" {
				return Policy{}, fmt.Errorf("line %d: fallback declaration is incomplete", lineNo)
			}
			policy.Fallback[Decision(pairs["status"])] = pairs["operation"]
		case "replay":
			pairs, err := pairsAfter(tokens, 1)
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			policy.Replay = PolicyRule{Name: pairs["rule"], Status: Decision(pairs["status"]), Requires: pairs["requires"]}
		case "metrics":
			pairs, err := pairsAfter(tokens, 1)
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			policy.Metrics.Vector = splitCSV(pairs["vector"])
			policy.Metrics.Pair = pairs["pair"]
			policy.Metrics.Improvement = pairs["improvement"]
			policy.Metrics.Forbidden = splitCSV(pairs["forbidden"])
		case "authority_rule":
			pairs, err := pairsAfter(tokens, 1)
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			policy.AuthorityRule.OutputScope = pairs["output_scope"]
			policy.AuthorityRule.RepositoryWrites, err = strconv.Atoi(pairs["repository_writes"])
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: malformed repository_writes", lineNo)
			}
			policy.AuthorityRule.AutomaticCommit, err = strconv.Atoi(pairs["automatic_commit"])
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: malformed automatic_commit", lineNo)
			}
			policy.AuthorityRule.AutomaticPush, err = strconv.Atoi(pairs["automatic_push"])
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: malformed automatic_push", lineNo)
			}
			policy.AuthorityRule.AutomaticMerge, err = strconv.Atoi(pairs["automatic_merge"])
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: malformed automatic_merge", lineNo)
			}
			policy.AuthorityRule.AutomaticRelease, err = strconv.Atoi(pairs["automatic_release"])
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: malformed automatic_release", lineNo)
			}
		case "generation":
			if len(tokens) < 2 {
				return Policy{}, fmt.Errorf("line %d: generation language is required", lineNo)
			}
			pairs, err := pairsAfter(tokens, 2)
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			policy.Generation = GenerationPlan{Language: tokens[1], Package: pairs["package"], Entrypoint: pairs["entrypoint"]}
		case "scenario":
			if len(tokens) < 3 {
				return Policy{}, fmt.Errorf("line %d: scenario id is required", lineNo)
			}
			pairs, err := pairsAfter(tokens, 2)
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: %w", lineNo, err)
			}
			replay, err := strconv.ParseBool(pairs["replay"])
			if err != nil {
				return Policy{}, fmt.Errorf("line %d: malformed replay flag", lineNo)
			}
			policy.Scenarios = append(policy.Scenarios, ScenarioSpec{ID: tokens[1], Expected: Decision(pairs["expected"]), Source: pairs["source"], CacheSource: pairs["cache_source"], Variant: pairs["variant"], Replay: replay})
		default:
			return Policy{}, fmt.Errorf("line %d: unknown contract record %q", lineNo, tokens[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return Policy{}, err
	}
	if !seenHeader || inside {
		return Policy{}, fmt.Errorf("incomplete contract declaration")
	}
	if err := ValidatePolicy(policy); err != nil {
		return Policy{}, err
	}
	return policy, nil
}

func ValidatePolicy(policy Policy) error {
	if policy.Schema == "" || policy.Authority != "metacode" {
		return fmt.Errorf("contract schema or authority is invalid")
	}
	if len(policy.Precedence) != 3 || policy.Precedence[0] != Refuted || policy.Precedence[1] != Unknown || policy.Precedence[2] != Closed {
		return fmt.Errorf("contract precedence is not REFUTED > UNKNOWN > CLOSED")
	}
	if len(policy.UnknownFields) != 6 || strings.Join(policy.UnknownFields, ",") != "stage,step,reason,unknown_class,next_operation,blocked_by" {
		return fmt.Errorf("contract UNKNOWN fields are incomplete")
	}
	if policy.Denominator.Count != 7 || len(policy.Scenarios) != policy.Denominator.Count {
		return fmt.Errorf("contract denominator and fixed cases disagree")
	}
	for _, key := range []string{"semantic_key", "dependency_binding", "origin_binding", "toolchain_binding", "contract_binding"} {
		if policy.CacheKey[key] == "" {
			return fmt.Errorf("cache key binding %q is missing", key)
		}
	}
	seen := map[string]bool{}
	for _, scenario := range policy.Scenarios {
		if scenario.ID == "" || seen[scenario.ID] || scenario.Source == "" || scenario.CacheSource == "" || scenario.Variant == "" {
			return fmt.Errorf("fixed scenario is incomplete or duplicated")
		}
		seen[scenario.ID] = true
		if scenario.Expected != Closed && scenario.Expected != Unknown && scenario.Expected != Refuted {
			return fmt.Errorf("scenario %q has invalid expected status", scenario.ID)
		}
	}
	if len(policy.Bindings) != 5 || len(policy.Obligations) != 5 || policy.Witness.Kind != "terminal" || policy.Reuse.Status != Closed || policy.Replay.Status != Closed {
		return fmt.Errorf("binding, proof, witness, reuse, or replay declarations are incomplete")
	}
	if len(policy.Metrics.Vector) != 4 || policy.Metrics.Pair == "" || policy.Metrics.Improvement == "" {
		return fmt.Errorf("metric vector declaration is incomplete")
	}
	if policy.AuthorityRule.RepositoryWrites != 0 || policy.AuthorityRule.OutputScope == "" || policy.AuthorityRule.AutomaticCommit != 0 || policy.AuthorityRule.AutomaticPush != 0 || policy.AuthorityRule.AutomaticMerge != 0 || policy.AuthorityRule.AutomaticRelease != 0 {
		return fmt.Errorf("authority boundary is not zero-write and caller-owned")
	}
	return nil
}

func LoadCorpus(path string) (Corpus, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Corpus{}, "", err
	}
	var corpus Corpus
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&corpus); err != nil {
		return Corpus{}, "", fmt.Errorf("decode corpus: %w", err)
	}
	if corpus.Schema == "" || len(corpus.Cases) == 0 {
		return Corpus{}, "", fmt.Errorf("corpus is empty")
	}
	return corpus, Digest(raw), nil
}

func WriteJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func Digest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func DigestFile(path string) (string, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	return Digest(raw), raw, nil
}

func tokenize(line string) ([]string, error) {
	var tokens []string
	for i := 0; i < len(line); {
		for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
			i++
		}
		if i == len(line) {
			break
		}
		if line[i] == '"' {
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
			continue
		}
		start := i
		for i < len(line) && line[i] != ' ' && line[i] != '\t' {
			i++
		}
		tokens = append(tokens, line[start:i])
	}
	return tokens, nil
}

func pairsAfter(tokens []string, start int) (map[string]string, error) {
	if start > len(tokens) || len(tokens[start:])%2 != 0 {
		return nil, fmt.Errorf("expected key/value pairs")
	}
	pairs := make(map[string]string, len(tokens[start:])/2)
	for i := start; i < len(tokens); i += 2 {
		if tokens[i] == "" || pairs[tokens[i]] != "" {
			return nil, fmt.Errorf("duplicate or malformed key %q", tokens[i])
		}
		pairs[tokens[i]] = tokens[i+1]
	}
	return pairs, nil
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			result = append(result, strings.TrimSpace(part))
		}
	}
	return result
}

func stripMetaComment(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") {
		return ""
	}
	if index := strings.Index(line, "//"); index >= 0 {
		return line[:index]
	}
	return line
}

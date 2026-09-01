package engine

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-proof-carrying-semantic-cache/internal/oracle"
	"github.com/kimjooyoon/gooo-proof-carrying-semantic-cache/internal/protocol"
)

type Options struct {
	Root         string
	MetaPath     string
	CorpusPath   string
	OutputPath   string
	Toolchain    string
	RunnerDigest string
}

func RunConformance(options Options) (protocol.ConformanceReport, error) {
	policy, contractDigest, err := protocol.LoadPolicy(options.MetaPath)
	if err != nil {
		return protocol.ConformanceReport{}, err
	}
	corpus, corpusDigest, err := protocol.LoadCorpus(options.CorpusPath)
	if err != nil {
		return protocol.ConformanceReport{}, err
	}
	if len(corpus.Cases) != policy.Denominator.Count {
		return protocol.ConformanceReport{}, fmt.Errorf("corpus count does not match metacode denominator")
	}
	if err := requireOutside(options.Root, options.OutputPath); err != nil {
		return protocol.ConformanceReport{}, err
	}
	if err := ensureEmpty(options.OutputPath); err != nil {
		return protocol.ConformanceReport{}, err
	}
	if err := os.MkdirAll(options.OutputPath, 0o755); err != nil {
		return protocol.ConformanceReport{}, err
	}
	if options.Toolchain == "" {
		options.Toolchain = runtime.Version() + "/" + runtime.GOOS + "/" + runtime.GOARCH
	}
	if options.RunnerDigest == "" {
		options.RunnerDigest = "sha256:runner-unspecified"
	}

	byID := make(map[string]protocol.FixtureCase, len(corpus.Cases))
	for _, fixture := range corpus.Cases {
		if _, exists := byID[fixture.ID]; exists {
			return protocol.ConformanceReport{}, fmt.Errorf("duplicate corpus case %q", fixture.ID)
		}
		byID[fixture.ID] = fixture
	}
	var report protocol.ConformanceReport
	report.Schema = "gooo-proof-carrying-semantic-cache/conformance/v1"
	report.ContractDigest = contractDigest
	report.CorpusDigest = corpusDigest
	report.Authority = policy.AuthorityRule
	report.Inventory = Inventory(options.Root)
	for _, fixed := range policy.Scenarios {
		fixture, ok := byID[fixed.ID]
		if !ok {
			return protocol.ConformanceReport{}, fmt.Errorf("missing corpus fixture for %q", fixed.ID)
		}
		caseReport, err := runCase(policy, contractDigest, options, fixed, fixture)
		if err != nil {
			return protocol.ConformanceReport{}, fmt.Errorf("case %s: %w", fixed.ID, err)
		}
		report.Cases = append(report.Cases, caseReport)
		report.Summary.TotalCases++
		switch caseReport.Decision {
		case protocol.Closed:
			report.Summary.Closed++
		case protocol.Unknown:
			report.Summary.Unknown++
		case protocol.Refuted:
			report.Summary.Refuted++
		}
		report.Summary.TestsTotal += caseReport.TestsTotal
		report.Summary.TestsSelected += caseReport.TestsSelected
		report.Summary.TestsExecuted += caseReport.TestsExecuted
		report.Summary.TestsReused += caseReport.TestsReused
		if caseReport.Decision == protocol.Refuted {
			report.Summary.TestsFailed++
		}
		if caseReport.Decision == protocol.Unknown {
			report.Summary.TestsUnknown++
		}
	}
	report.Decision = protocol.Closed
	for index, fixed := range policy.Scenarios {
		if report.Cases[index].Decision != fixed.Expected {
			report.Decision = resolve(report.Cases[index].Decision, report.Decision, policy.Precedence)
		}
	}
	if report.Decision == "" {
		report.Decision = protocol.Closed
	}
	return report, nil
}

func runCase(policy protocol.Policy, contractDigest string, options Options, fixed protocol.ScenarioSpec, fixture protocol.FixtureCase) (protocol.CaseReport, error) {
	currentPath := filepath.Join(options.Root, fixed.Source)
	cachePath := filepath.Join(options.Root, fixed.CacheSource)
	currentRaw, err := os.ReadFile(currentPath)
	if err != nil {
		return protocol.CaseReport{}, err
	}
	cacheRaw, err := os.ReadFile(cachePath)
	if err != nil {
		return protocol.CaseReport{}, err
	}
	currentToolchainBinding := protocol.Digest([]byte(options.Toolchain + "|" + options.RunnerDigest))
	currentDependencies := fixture.Dependencies
	cacheDependencies := fixture.CacheDependencies
	if len(cacheDependencies) == 0 {
		cacheDependencies = currentDependencies
	}
	currentDependencyBinding := dependencyBinding(currentDependencies)
	cacheDependencyBinding := dependencyBinding(cacheDependencies)
	originBinding := protocol.Digest([]byte(fixture.Origin))
	cacheToolchainBinding := currentToolchainBinding
	if fixture.CacheToolchain != "" {
		cacheToolchainBinding = fixture.CacheToolchain
	}

	coldStart := time.Now()
	currentIR, currentArtifact, err := oracle.Rebuild(fixed.Source, currentRaw, policy)
	if err != nil {
		return protocol.CaseReport{}, err
	}
	coldVector := protocol.MetricVector{TestsExecuted: 1, TestsReused: 0, WallMS: measuredMS(coldStart), PeakRSSKiB: peakRSSKiB()}

	cacheIR, cacheArtifact, err := oracle.Rebuild(fixed.CacheSource, cacheRaw, policy)
	if err != nil {
		return protocol.CaseReport{}, err
	}
	cacheEntry := protocol.CacheEntry{
		Schema: "gooo/proof-carrying-semantic-cache/entry/v1",
		Key: protocol.CacheKey{
			SemanticKey:       cacheIR.SemanticKey,
			DependencyBinding: cacheDependencyBinding,
			OriginBinding:     originBinding,
			ToolchainBinding:  cacheToolchainBinding,
			ContractBinding:   contractDigest,
		},
		Artifact: cacheArtifact,
	}
	if fixture.CacheProofPresent {
		terminal := cacheIR.Terminal
		if fixture.CacheTerminalReason != "" {
			terminal.Reason = fixture.CacheTerminalReason
		}
		if fixture.CacheTerminalEffect != "" {
			terminal.Effect = fixture.CacheTerminalEffect
		}
		obligations := make(map[string]string, len(policy.Obligations))
		for _, obligation := range policy.Obligations {
			obligations[obligation.ID] = obligation.Proof
		}
		cacheEntry.Proof = &protocol.ProofReceipt{
			Schema:            "gooo/proof-carrying-semantic-cache/proof-receipt/v1",
			SemanticKey:       cacheIR.SemanticKey,
			DependencyBinding: cacheDependencyBinding,
			OriginBinding:     originBinding,
			ToolchainBinding:  cacheToolchainBinding,
			ContractBinding:   contractDigest,
			ArtifactDigest:    cacheArtifact.Digest,
			Obligations:       obligations,
			Terminal:          terminal,
			IndependentOracle: cacheArtifact.Digest,
		}
	}

	warmStart := time.Now()
	unknowns, refutations := verifyCache(policy, currentIR, currentArtifact, cacheEntry, currentDependencyBinding, originBinding, currentToolchainBinding, contractDigest)
	decision := resolveEvidence(unknowns, refutations, policy.Precedence)
	warmTestsExecuted, warmTestsReused := 1, 0
	if decision == protocol.Closed {
		warmTestsExecuted, warmTestsReused = 0, 1
	}
	warmVector := protocol.MetricVector{TestsExecuted: warmTestsExecuted, TestsReused: warmTestsReused, WallMS: measuredMS(warmStart), PeakRSSKiB: peakRSSKiB()}

	replay := protocol.ReplayObservation{}
	if fixed.Replay {
		replayStart := time.Now()
		replayIR, replayArtifact, replayErr := oracle.Rebuild(fixed.Source, currentRaw, policy)
		if replayErr != nil {
			return protocol.CaseReport{}, replayErr
		}
		replay = protocol.ReplayObservation{
			SemanticKeySame: replayIR.SemanticKey == currentIR.SemanticKey,
			ArtifactSame:    bytes.Equal(replayArtifact.Bytes, currentArtifact.Bytes),
			TerminalSame:    replayIR.Terminal == currentIR.Terminal,
			WallMS:          measuredMS(replayStart),
		}
		if !replay.SemanticKeySame || !replay.ArtifactSame || !replay.TerminalSame {
			refutations = append(refutations, protocol.Refutation{Kind: "REPLAY", Reason: "REPLAY_IDENTITY_MISMATCH"})
			decision = resolveEvidence(unknowns, refutations, policy.Precedence)
		}
	}

	claim := protocol.ImprovementClaim{Status: protocol.Unknown, Reason: "NO_PROVED_REUSE", ExactPair: false, Before: coldVector, After: warmVector}
	if decision == protocol.Closed && warmTestsReused == 1 && options.RunnerDigest != "" {
		delta := warmVector.WallMS - coldVector.WallMS
		claim = protocol.ImprovementClaim{Status: protocol.Closed, Reason: "EXACT_PAIR_AND_INDEPENDENT_ORACLE_CLOSED", ExactPair: true, Before: coldVector, After: warmVector, WallMSDelta: &delta}
	}
	report := protocol.CaseReport{
		ID: fixed.ID, Expected: fixed.Expected, Decision: decision,
		Reason:          decisionReason(decision, unknowns, refutations),
		SourceRawDigest: currentIR.RawDigest, SemanticKey: currentIR.SemanticKey,
		DependencyBinding: currentDependencyBinding, OriginBinding: originBinding,
		ToolchainBinding: currentToolchainBinding, ContractBinding: contractDigest,
		CacheHit: true, RebuildPerformed: decision != protocol.Closed,
		IndependentOracle: currentArtifact.Digest, CacheEntry: cacheEntry,
		Unknowns: unknowns, Refutations: refutations, Terminal: currentIR.Terminal,
		OracleTerminal: currentIR.Terminal, Pair: protocol.PairVector{Cold: coldVector, ProvedReuse: warmVector, ExactPair: claim.ExactPair},
		Improvement: claim, TestID: fixture.TestID, TestsTotal: 1, TestsSelected: 1,
		TestsExecuted: warmTestsExecuted, TestsReused: warmTestsReused, Replay: replay,
		PairIdentity: protocol.PairIdentity{ScenarioID: fixed.ID, SourceDigest: currentIR.RawDigest, SemanticKey: currentIR.SemanticKey, ContractDigest: contractDigest, ToolchainBinding: currentToolchainBinding, RunnerDigest: options.RunnerDigest},
	}
	caseDir := filepath.Join(options.OutputPath, fixed.ID)
	if err := os.MkdirAll(filepath.Join(caseDir, "generated"), 0o755); err != nil {
		return protocol.CaseReport{}, err
	}
	if err := os.WriteFile(filepath.Join(caseDir, "generated", "main.go"), currentArtifact.Bytes, 0o644); err != nil {
		return protocol.CaseReport{}, err
	}
	if err := protocol.WriteJSON(filepath.Join(caseDir, "cache-entry.json"), cacheEntry); err != nil {
		return protocol.CaseReport{}, err
	}
	if err := protocol.WriteJSON(filepath.Join(caseDir, "report.json"), report); err != nil {
		return protocol.CaseReport{}, err
	}
	if err := protocol.WriteJSON(filepath.Join(caseDir, "oracle-ir.json"), currentIR); err != nil {
		return protocol.CaseReport{}, err
	}
	if err := protocol.WriteJSON(filepath.Join(caseDir, "replay.json"), replay); err != nil {
		return protocol.CaseReport{}, err
	}
	return report, nil
}

func verifyCache(policy protocol.Policy, currentIR protocol.SemanticIR, currentArtifact protocol.GeneratedArtifact, cache protocol.CacheEntry, dependencyBinding, originBinding, toolchainBinding, contractDigest string) ([]protocol.UnknownEvidence, []protocol.Refutation) {
	var unknowns []protocol.UnknownEvidence
	var refutations []protocol.Refutation
	blocked := map[string]bool{}
	for _, binding := range policy.Bindings {
		current, cached := bindingValue(binding.Kind, currentIR, cache, dependencyBinding, originBinding, toolchainBinding, contractDigest)
		if current == cached {
			continue
		}
		blocked[binding.Kind] = true
		if binding.Mismatch == protocol.Refuted {
			refutations = append(refutations, protocol.Refutation{Kind: strings.ToUpper(binding.Kind), Reason: binding.Kind + "_BINDING_MISMATCH"})
			continue
		}
		unknowns = append(unknowns, makeUnknown(binding.Stage, binding.Step, binding.Kind+" binding differs across the cache boundary", binding.UnknownClass, fallbackOperation(policy, protocol.Unknown), []string{binding.Kind + "_binding"}))
	}
	if cache.Proof == nil {
		obligation := policy.Obligations[0]
		unknowns = append(unknowns, makeUnknown(obligation.Stage, obligation.Step, obligation.Missing, "PROOF_RECEIPT_MISSING", fallbackOperation(policy, protocol.Unknown), []string{"proof_receipt"}))
		return unknowns, refutations
	}
	proof := cache.Proof
	if !blocked["dependency"] && proof.DependencyBinding != dependencyBinding {
		refutations = append(refutations, protocol.Refutation{Kind: "PROOF", Reason: "DEPENDENCY_PROOF_MISMATCH"})
	}
	if !blocked["origin"] && proof.OriginBinding != originBinding {
		refutations = append(refutations, protocol.Refutation{Kind: "PROOF", Reason: "ORIGIN_PROOF_MISMATCH"})
	}
	if !blocked["toolchain"] && proof.ToolchainBinding != toolchainBinding {
		refutations = append(refutations, protocol.Refutation{Kind: "PROOF", Reason: "TOOLCHAIN_PROOF_MISMATCH"})
	}
	if !blocked["contract"] && proof.ContractBinding != contractDigest {
		refutations = append(refutations, protocol.Refutation{Kind: "PROOF", Reason: "CONTRACT_PROOF_MISMATCH"})
	}
	if !blocked["semantic_key"] && proof.SemanticKey != currentIR.SemanticKey {
		refutations = append(refutations, protocol.Refutation{Kind: "PROOF", Reason: "SEMANTIC_KEY_PROOF_MISMATCH"})
	}
	for _, obligation := range policy.Obligations {
		value, ok := proof.Obligations[obligation.ID]
		if !ok || value == "" {
			unknowns = append(unknowns, makeUnknown(obligation.Stage, obligation.Step, obligation.Missing, "PROOF_OBLIGATION_MISSING", fallbackOperation(policy, protocol.Unknown), []string{obligation.ID}))
			continue
		}
		if value != obligation.Proof {
			refutations = append(refutations, protocol.Refutation{Kind: "PROOF", Reason: obligation.ID + "_PROOF_CONTRADICTED"})
		}
	}
	if proof.Terminal != currentIR.Terminal {
		refutations = append(refutations, protocol.Refutation{Kind: "WITNESS", Reason: "TERMINAL_REASON_OR_EFFECT_MISMATCH"})
	}
	if proof.ArtifactDigest != cache.Artifact.Digest || protocol.Digest(cache.Artifact.Bytes) != cache.Artifact.Digest {
		refutations = append(refutations, protocol.Refutation{Kind: "ARTIFACT", Reason: "CACHE_ARTIFACT_DIGEST_MISMATCH"})
	}
	if cache.Artifact.Digest != currentArtifact.Digest {
		refutations = append(refutations, protocol.Refutation{Kind: "ORACLE", Reason: "INDEPENDENT_REBUILD_ORACLE_MISMATCH"})
	}
	if proof.IndependentOracle != currentArtifact.Digest {
		refutations = append(refutations, protocol.Refutation{Kind: "ORACLE", Reason: "PROOF_ORACLE_DIGEST_MISMATCH"})
	}
	return unknowns, refutations
}

func bindingValue(kind string, current protocol.SemanticIR, cache protocol.CacheEntry, dependencyBinding, originBinding, toolchainBinding, contractDigest string) (string, string) {
	switch kind {
	case "dependency":
		return dependencyBinding, cache.Key.DependencyBinding
	case "origin":
		return originBinding, cache.Key.OriginBinding
	case "toolchain":
		return toolchainBinding, cache.Key.ToolchainBinding
	case "contract":
		return contractDigest, cache.Key.ContractBinding
	default:
		return current.SemanticKey, cache.Key.SemanticKey
	}
}

func dependencyBinding(dependencies []protocol.Dependency) string {
	copyOf := append([]protocol.Dependency(nil), dependencies...)
	sort.Slice(copyOf, func(i, j int) bool { return copyOf[i].Name < copyOf[j].Name })
	var builder strings.Builder
	for _, dependency := range copyOf {
		builder.WriteString(dependency.Name)
		builder.WriteByte('=')
		builder.WriteString(dependency.Digest)
		builder.WriteByte('\n')
	}
	return protocol.Digest([]byte(builder.String()))
}

func resolveEvidence(unknowns []protocol.UnknownEvidence, refutations []protocol.Refutation, precedence []protocol.Decision) protocol.Decision {
	if len(refutations) > 0 && contains(precedence, protocol.Refuted) {
		return protocol.Refuted
	}
	if len(unknowns) > 0 && contains(precedence, protocol.Unknown) {
		return protocol.Unknown
	}
	return protocol.Closed
}

func resolve(actual, fallback protocol.Decision, precedence []protocol.Decision) protocol.Decision {
	if actual == fallback {
		return actual
	}
	for _, value := range precedence {
		if actual == value {
			return actual
		}
		if fallback == value {
			return fallback
		}
	}
	return protocol.Unknown
}

func contains(values []protocol.Decision, wanted protocol.Decision) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func makeUnknown(stage, step, reason, class, nextOperation string, blockedBy []string) protocol.UnknownEvidence {
	return protocol.UnknownEvidence{Stage: stage, Step: step, Reason: reason, UnknownClass: class, NextOperation: nextOperation, BlockedBy: blockedBy}
}

func fallbackOperation(policy protocol.Policy, status protocol.Decision) string {
	if operation := policy.Fallback[status]; operation != "" {
		return operation
	}
	return "independent-rebuild"
}

func decisionReason(decision protocol.Decision, unknowns []protocol.UnknownEvidence, refutations []protocol.Refutation) string {
	if decision == protocol.Refuted && len(refutations) > 0 {
		return refutations[0].Reason
	}
	if decision == protocol.Unknown && len(unknowns) > 0 {
		return unknowns[0].Reason
	}
	return "all bindings, proof obligations, terminal witness, and independent oracle matched"
}

func measuredMS(start time.Time) int64 {
	value := time.Since(start).Milliseconds()
	if value < 1 {
		return 1
	}
	return value
}

func peakRSSKiB() int64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return int64(stats.Sys / 1024)
}

func Inventory(root string) protocol.Inventory {
	var inventory protocol.Inventory
	inventory.RootReadmeExcluded = true
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if path == filepath.Join(root, ".git") {
			return filepath.SkipDir
		}
		if info.IsDir() {
			if path != root {
				inventory.DescendantDirs++
			}
			return nil
		}
		if filepath.Base(path) == "README.md" && filepath.Dir(path) == root {
			return nil
		}
		inventory.RegularFiles++
		switch filepath.Ext(path) {
		case ".go":
			inventory.GoFiles++
			inventory.GoPhysicalLines += lineCount(path)
		case ".gooo":
			inventory.GoooFiles++
			inventory.GoooPhysicalLines += lineCount(path)
		}
		return nil
	})
	return inventory
}

func lineCount(path string) int64 {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	if len(raw) == 0 {
		return 0
	}
	count := int64(bytes.Count(raw, []byte{'\n'}))
	if raw[len(raw)-1] != '\n' {
		count++
	}
	return count
}

func requireOutside(root, output string) error {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absoluteOutput, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(absoluteRoot, absoluteOutput)
	if err != nil {
		return err
	}
	if relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return fmt.Errorf("output must be caller-owned and outside repository")
	}
	return nil
}

func ensureEmpty(path string) error {
	entries, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("output directory must be empty")
	}
	return nil
}

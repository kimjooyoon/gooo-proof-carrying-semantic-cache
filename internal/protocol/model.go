package protocol

import "encoding/json"

type Decision string

const (
	Closed  Decision = "CLOSED"
	Unknown Decision = "UNKNOWN"
	Refuted Decision = "REFUTED"
)

type Policy struct {
	Schema        string
	Authority     string
	Precedence    []Decision
	UnknownFields []string
	Denominator   Denominator
	Normalization map[string]string
	CacheKey      map[string]string
	Bindings      []BindingRule
	Obligations   []ProofObligation
	Witness       WitnessRule
	Reuse         PolicyRule
	Fallback      map[Decision]string
	Replay        PolicyRule
	Metrics       MetricPolicy
	AuthorityRule AuthorityPolicy
	Generation    GenerationPlan
	Scenarios     []ScenarioSpec
}

type Denominator struct {
	ID    string
	Count int
}

type BindingRule struct {
	Kind         string
	Stage        string
	Step         string
	Mismatch     Decision
	UnknownClass string
}

type ProofObligation struct {
	ID      string
	Stage   string
	Step    string
	Proof   string
	Missing string
}

type WitnessRule struct {
	Kind     string
	Stage    string
	Step     string
	Reason   string
	Effect   string
	Mismatch Decision
}

type PolicyRule struct {
	Name     string
	Status   Decision
	Requires string
}

type MetricPolicy struct {
	Vector      []string
	Pair        string
	Improvement string
	Forbidden   []string
}

type AuthorityPolicy struct {
	RepositoryWrites int    `json:"repository_writes"`
	OutputScope      string `json:"output_scope"`
	AutomaticCommit  int    `json:"automatic_commit"`
	AutomaticPush    int    `json:"automatic_push"`
	AutomaticMerge   int    `json:"automatic_merge"`
	AutomaticRelease int    `json:"automatic_release"`
}

type GenerationPlan struct {
	Language   string
	Package    string
	Entrypoint string
}

type ScenarioSpec struct {
	ID          string
	Expected    Decision
	Source      string
	CacheSource string
	Variant     string
	Replay      bool
}

type Corpus struct {
	Schema   string        `json:"schema"`
	CorpusID string        `json:"corpus_id"`
	Cases    []FixtureCase `json:"cases"`
}

type FixtureCase struct {
	ID                  string       `json:"id"`
	TestID              string       `json:"test_id"`
	Origin              string       `json:"origin"`
	Dependencies        []Dependency `json:"dependencies"`
	CacheDependencies   []Dependency `json:"cache_dependencies,omitempty"`
	CacheProofPresent   bool         `json:"cache_proof_present"`
	CacheToolchain      string       `json:"cache_toolchain,omitempty"`
	CacheTerminalReason string       `json:"cache_terminal_reason,omitempty"`
	CacheTerminalEffect string       `json:"cache_terminal_effect,omitempty"`
}

type Dependency struct {
	Name   string `json:"name"`
	Digest string `json:"digest"`
}

type Program struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
	Effect string `json:"effect"`
}

type TerminalWitness struct {
	Reason string `json:"reason"`
	Effect string `json:"effect"`
}

type SemanticIR struct {
	Schema      string          `json:"schema"`
	SourcePath  string          `json:"source_path"`
	RawDigest   string          `json:"raw_digest"`
	SemanticKey string          `json:"semantic_key"`
	Canonical   string          `json:"canonical"`
	Program     Program         `json:"program"`
	Terminal    TerminalWitness `json:"terminal"`
}

type GeneratedArtifact struct {
	Path   string `json:"path"`
	Bytes  []byte `json:"bytes"`
	Digest string `json:"digest"`
}

type ProofReceipt struct {
	Schema            string            `json:"schema"`
	SemanticKey       string            `json:"semantic_key"`
	DependencyBinding string            `json:"dependency_binding"`
	OriginBinding     string            `json:"origin_binding"`
	ToolchainBinding  string            `json:"toolchain_binding"`
	ContractBinding   string            `json:"contract_binding"`
	ArtifactDigest    string            `json:"artifact_digest"`
	Obligations       map[string]string `json:"obligations"`
	Terminal          TerminalWitness   `json:"terminal"`
	IndependentOracle string            `json:"independent_oracle_digest"`
}

type CacheKey struct {
	SemanticKey       string `json:"semantic_key"`
	DependencyBinding string `json:"dependency_binding"`
	OriginBinding     string `json:"origin_binding"`
	ToolchainBinding  string `json:"toolchain_binding"`
	ContractBinding   string `json:"contract_binding"`
}

type CacheEntry struct {
	Schema   string            `json:"schema"`
	Key      CacheKey          `json:"key"`
	Artifact GeneratedArtifact `json:"generated_artifact"`
	Proof    *ProofReceipt     `json:"proof_receipt,omitempty"`
}

type UnknownEvidence struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type Refutation struct {
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
}

type MetricVector struct {
	TestsExecuted int   `json:"tests_executed"`
	TestsReused   int   `json:"tests_reused"`
	WallMS        int64 `json:"wall_ms"`
	PeakRSSKiB    int64 `json:"peak_rss_kib"`
}

type PairVector struct {
	Cold        MetricVector `json:"cold"`
	ProvedReuse MetricVector `json:"proved_reuse"`
	ExactPair   bool         `json:"exact_pair"`
}

type PairIdentity struct {
	ScenarioID       string `json:"scenario_id"`
	SourceDigest     string `json:"source_digest"`
	SemanticKey      string `json:"semantic_key"`
	ContractDigest   string `json:"contract_digest"`
	ToolchainBinding string `json:"toolchain_binding"`
	RunnerDigest     string `json:"runner_digest"`
}

type ReplayObservation struct {
	SemanticKeySame bool  `json:"semantic_key_same"`
	ArtifactSame    bool  `json:"artifact_same"`
	TerminalSame    bool  `json:"terminal_same"`
	WallMS          int64 `json:"wall_ms"`
}

type ImprovementClaim struct {
	Status      Decision     `json:"status"`
	Reason      string       `json:"reason"`
	ExactPair   bool         `json:"exact_pair"`
	Before      MetricVector `json:"before"`
	After       MetricVector `json:"after"`
	WallMSDelta *int64       `json:"wall_ms_delta,omitempty"`
}

type StageMeasurement struct {
	WallMS     int64 `json:"wall_ms"`
	PeakRSSKiB int64 `json:"peak_rss_kib"`
}

type StageMeasurements struct {
	Compile     StageMeasurement `json:"compile"`
	Build       StageMeasurement `json:"build"`
	Test        StageMeasurement `json:"test"`
	Conformance StageMeasurement `json:"conformance"`
	Integration StageMeasurement `json:"integration"`
}

type Inventory struct {
	GoFiles            int64 `json:"go_files"`
	GoooFiles          int64 `json:"gooo_files"`
	GoPhysicalLines    int64 `json:"go_physical_lines"`
	GoooPhysicalLines  int64 `json:"gooo_physical_lines"`
	DescendantDirs     int64 `json:"descendant_dirs"`
	RegularFiles       int64 `json:"regular_files"`
	RootReadmeExcluded bool  `json:"root_readme_excluded"`
}

type CaseReport struct {
	ID                string            `json:"id"`
	Expected          Decision          `json:"expected"`
	Decision          Decision          `json:"decision"`
	Reason            string            `json:"reason"`
	SourceRawDigest   string            `json:"source_raw_digest"`
	SemanticKey       string            `json:"semantic_key"`
	DependencyBinding string            `json:"dependency_binding"`
	OriginBinding     string            `json:"origin_binding"`
	ToolchainBinding  string            `json:"toolchain_binding"`
	ContractBinding   string            `json:"contract_binding"`
	CacheHit          bool              `json:"cache_hit"`
	RebuildPerformed  bool              `json:"rebuild_performed"`
	IndependentOracle string            `json:"independent_oracle_digest"`
	CacheEntry        CacheEntry        `json:"cache_entry"`
	Unknowns          []UnknownEvidence `json:"unknowns"`
	Refutations       []Refutation      `json:"refutations"`
	Terminal          TerminalWitness   `json:"terminal"`
	OracleTerminal    TerminalWitness   `json:"oracle_terminal"`
	Pair              PairVector        `json:"pair"`
	PairIdentity      PairIdentity      `json:"pair_identity"`
	Improvement       ImprovementClaim  `json:"improvement"`
	Replay            ReplayObservation `json:"replay"`
	TestID            string            `json:"test_id"`
	TestsTotal        int               `json:"tests_total"`
	TestsSelected     int               `json:"tests_selected"`
	TestsExecuted     int               `json:"tests_executed"`
	TestsReused       int               `json:"tests_reused"`
}

type ConformanceSummary struct {
	TotalCases    int `json:"total_cases"`
	Closed        int `json:"closed"`
	Unknown       int `json:"unknown"`
	Refuted       int `json:"refuted"`
	TestsTotal    int `json:"tests_total"`
	TestsSelected int `json:"tests_selected"`
	TestsExecuted int `json:"tests_executed"`
	TestsReused   int `json:"tests_reused"`
	TestsFailed   int `json:"tests_failed"`
	TestsUnknown  int `json:"tests_unknown"`
}

type ConformanceReport struct {
	Schema            string             `json:"schema"`
	ContractDigest    string             `json:"contract_digest"`
	CorpusDigest      string             `json:"corpus_digest"`
	Decision          Decision           `json:"decision"`
	Cases             []CaseReport       `json:"cases"`
	Summary           ConformanceSummary `json:"summary"`
	Inventory         Inventory          `json:"inventory"`
	StageMeasurements StageMeasurements  `json:"stage_measurements"`
	Authority         AuthorityPolicy    `json:"authority"`
}

func (r ConformanceReport) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

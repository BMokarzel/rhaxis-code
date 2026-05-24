package node

type ManifestType string

const (
	ManifestGoMod    ManifestType = "go.mod"
	ManifestNpm      ManifestType = "package.json"
	ManifestPyProj   ManifestType = "pyproject.toml"
	ManifestPySetup  ManifestType = "setup.py"
	ManifestCargo    ManifestType = "Cargo.toml"
	ManifestPom      ManifestType = "pom.xml"
	ManifestGradle   ManifestType = "build.gradle"
	ManifestComposer ManifestType = "composer.json"
	ManifestGemfile  ManifestType = "Gemfile"
	ManifestNuget    ManifestType = "*.csproj"
	ManifestUnknown  ManifestType = "unknown"
)

type Service struct {
	Node
	Repo                 string `json:"repo"`               // canonical repo name
	Language             string `json:"language"`           // "go" | "python" | "typescript" | ...
	Manifest             string `json:"manifest,omitempty"` // caminho do manifest no repo
	ExternalDependencies []string
	FeatureTags          []string          `json:"feature_tags,omitempty"` // F-028 (ADR-009)
	Tags                 map[string]string `json:"tags,omitempty"`
}

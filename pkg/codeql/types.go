package codeql

// Types
type CodeqlCli struct {
	Path string
}

type RunQueryResult struct {
	ResultCount          int
	DatabaseSHA          string
	SourceLocationPrefix string
	BqrsFilePaths        BqrsFilePaths
	SarifFilePath        string
}

type BqrsFilePaths struct {
	BasePath          string   `json:"basePath"`
	RelativeFilePaths []string `json:"relativeFilePaths"`
}

type SarifOutputType string

const (
	Problem     SarifOutputType = "problem"
	PathProblem SarifOutputType = "path-problem"
)

type SarifRun struct {
	// XX: static types
	VersionControlProvenance []interface{} `json:"versionControlProvenance,omitempty"`
	// XX: never set, only read
	Results []interface{} `json:"results"`
}

type Sarif struct {
	Runs []SarifRun `json:"runs"`
}

type CreationMetadata struct {
	SHA        *string `yaml:"sha,omitempty"`
	CLIVersion *string `yaml:"cliVersion,omitempty"`
}

type DatabaseMetadata struct {
	CreationMetadata *CreationMetadata `yaml:"creationMetadata,omitempty"`
}

type QueryMetadata struct {
	ID   *string `json:"id,omitempty"`
	Kind *string `json:"kind,omitempty"`
}

type ResultSet struct {
	Name string `json:"name"`
	Rows int    `json:"rows"`
}

type BQRSInfo struct {
	ResultSets           []ResultSet `json:"resultSets"`
	CompatibleQueryKinds []string    `json:"compatibleQueryKinds"`
}

type Query struct {
	QueryPath            string        `json:"queryPath"`
	QueryMetadata        QueryMetadata `json:"queryMetadata"`
	RelativeBqrsFilePath string        `json:"relativeBqrsFilePath"`
	BqrsInfo             BQRSInfo      `json:"bqrsInfo"`
}

type QueryPackRunResults struct {
	Queries           []Query `json:"queries"`
	TotalResultsCount int     `json:"totalResultsCount"`
	ResultsBasePath   string  `json:"resultsBasePath"`
}

type ResolvedDatabase struct {
	SourceLocationPrefix string `json:"sourceLocationPrefix"`
}

type CodeQLCommandOutput struct {
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
}

package codeql

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"mrvacommander/pkg/queue"
	"mrvacommander/utils"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"gopkg.in/yaml.v3"
)

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getCodeQLCLIPath() (string, error) {
	// get the CODEQL_CLI_PATH environment variable
	codeqlCliPath := os.Getenv("CODEQL_CLI_PATH")
	if codeqlCliPath == "" {
		return "", fmt.Errorf("CODEQL_CLI_PATH environment variable not set")
	}
	return codeqlCliPath, nil
}

func GenerateResultsZipArchive(runQueryResult *RunQueryResult) ([]byte, error) {
	buffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buffer)

	if runQueryResult.SarifFilePath != "" {
		err := addFileToZip(zipWriter, runQueryResult.SarifFilePath, "results.sarif")
		if err != nil {
			return nil, fmt.Errorf("failed to add SARIF file to zip: %v", err)
		}
	}

	for _, relativePath := range runQueryResult.BqrsFilePaths.RelativeFilePaths {
		fullPath := filepath.Join(runQueryResult.BqrsFilePaths.BasePath, relativePath)
		err := addFileToZip(zipWriter, fullPath, relativePath)
		if err != nil {
			return nil, fmt.Errorf("failed to add BQRS file to zip: %v", err)
		}
	}

	err := zipWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close zip writer: %v", err)
	}

	return buffer.Bytes(), nil
}

func addFileToZip(zipWriter *zip.Writer, filePath, zipPath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	w, err := zipWriter.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip entry for %s: %v", zipPath, err)
	}

	_, err = io.Copy(w, file)
	if err != nil {
		return fmt.Errorf("failed to copy file content to zip entry for %s: %v", zipPath, err)
	}

	return nil
}

func RunQuery(database string, language queue.QueryLanguage,
	queryPackPath string, tempDir string) (*RunQueryResult, error) {
	path, err := getCodeQLCLIPath()

	if err != nil {
		return nil, fmt.Errorf("failed to get codeql cli path: %v", err)
	}

	codeql := CodeqlCli{path}

	resultsDir := filepath.Join(tempDir, "results")
	if err = os.Mkdir(resultsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create results directory: %v", err)
	}

	databasePath := filepath.Join(tempDir, "db")
	if utils.UnzipFile(database, databasePath) != nil {
		return nil, fmt.Errorf("failed to unzip database: %v", err)
	}

	dbMetadata, err := getDatabaseMetadata(databasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get database metadata: %v", err)
	}

	// Check if the database has CreationMetadata / a SHA
	var databaseSHA string
	if dbMetadata.CreationMetadata == nil || dbMetadata.CreationMetadata.SHA == nil {
		// If the database does not have a SHA, we can proceed regardless
		slog.Warn("Database does not have a SHA")
		databaseSHA = ""
	} else {
		databaseSHA = *dbMetadata.CreationMetadata.SHA
	}

	dbDir, err := findDBDir(databasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get database path: %v", err)
	}

	cmd := exec.Command(codeql.Path, "database", "run-queries",
		"--ram=2048", "--additional-packs", queryPackPath, "--", dbDir, queryPackPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		awaitSignal("XX: RunQuery: ", string(output))
		return nil, fmt.Errorf("failed to run queries: %v\nOutput: %s", err, output)
	}

	queryPackRunResults, err := getQueryPackRunResults(codeql, dbDir, queryPackPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get query pack run results: %v", err)
	}

	sourceLocationPrefix, err := getSourceLocationPrefix(codeql, dbDir)
	if err != nil {
		awaitSignal("XX: RunQuery ", "failed to get source location prefix ", "databasePath=", dbDir)
		return nil, fmt.Errorf("failed to get source location prefix: %v", err)
	}

	shouldGenerateSarif := queryPackSupportsSarif(queryPackRunResults)

	if shouldGenerateSarif {
		slog.Debug("Query pack supports SARIF")
	} else {
		slog.Debug("Query pack does not support SARIF")
	}

	var resultCount int
	var sarifFilePath string

	if shouldGenerateSarif {
		sarif, err := generateSarif(codeql, language, dbDir, queryPackPath, databaseSHA, resultsDir)
		if err != nil {
			return nil, fmt.Errorf("failed to generate SARIF: %v", err)
		}
		resultCount = getSarifResultCount(sarif)
		slog.Debug("Generated SARIF", "resultCount", resultCount)
		sarifFilePath = filepath.Join(resultsDir, "results.sarif")
		if err := os.WriteFile(sarifFilePath, sarif, 0644); err != nil {
			return nil, fmt.Errorf("failed to write SARIF file: %v", err)
		}
	} else {
		resultCount = queryPackRunResults.TotalResultsCount
		slog.Debug("Did not generate SARIF", "resultCount", resultCount)
	}

	slog.Debug("Adjusting BQRS files")
	bqrsFilePaths, err := adjustBqrsFiles(queryPackRunResults, resultsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to adjust BQRS files: %v", err)
	}

	return &RunQueryResult{
		ResultCount:          resultCount,
		DatabaseSHA:          databaseSHA,
		SourceLocationPrefix: sourceLocationPrefix,
		BqrsFilePaths:        bqrsFilePaths,
		SarifFilePath:        sarifFilePath,
	}, nil
}

//	awaitSignal()
//
// Debugging support function.  Freezes the current goroutine / function and
// waits for an external signal before continuing.
// Use
//
//	pkill -SIGUSR1 mrva_agent
//
// in the container to continue
func awaitSignal(strs ...string) {
	msg := strings.Join(strs, "")
	sigChan := make(chan os.Signal, 1)

	// Notify the channel on SIGUSR1 or SIGUSR2
	signal.Notify(sigChan, syscall.SIGUSR1, syscall.SIGUSR2)

	// Pause the program and wait for a signal
	slog.Debug(msg, "Waiting for SIGUSR1 or SIGUSR2...", nil)
	sig := <-sigChan

	// Handle the signal
	slog.Debug("Received signal: %s", sig.String(), nil)
}

//	findDBDir(rootdir)
//
// Find a subdirectory of `rootdir` that contains the files `codeql-database.yml`
// and `src.zip` and return its absolute path
func findDBDir(rootdir string) (string, error) {
	var dbDir string
	err := filepath.Walk(rootdir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if it's a directory
		if info.IsDir() {
			codeqlPath := filepath.Join(path, "codeql-database.yml")
			srcZipPath := filepath.Join(path, "src.zip")

			// Check if both files exist in this directory
			if _, err := os.Stat(codeqlPath); err == nil {
				if _, err := os.Stat(srcZipPath); err == nil {
					dbDir = path
					// Stop walking once we've found the directory
					return filepath.SkipDir
				}
			}
		}
		return nil
	})

	if err != nil {
		slog.Error("findDBDir: Problem in traversing directory:", "rootdir",
			rootdir, "err", err)
		return "", err
	}

	if dbDir == "" {
		slog.Error("Unable to find CodeQL DB directory in database zip", "rootdir", rootdir)
		return "", errors.New("no directory containing both 'codeql-database.yml' and 'src.zip' found")
	}

	return dbDir, nil
}

// Recursively search for files matching the glob pattern starting at rootdir
func globRecursively(rootdir string, pattern string) ([]string, error) {
	var matches []string

	// Walk the directory tree
	err := filepath.Walk(rootdir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if the file name matches the glob pattern
		match, err := filepath.Match(pattern, info.Name())
		if err != nil {
			return err
		}

		if !info.IsDir() && match {
			matches = append(matches, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return matches, nil
}

func getDatabaseMetadata(databasePath string) (*DatabaseMetadata, error) {
	paths, err := globRecursively(databasePath, "codeql-database.yml")
	if err != nil {
		return nil, fmt.Errorf("Failed to find codeql-database.yml %v", err)
	}
	if len(paths) != 1 {
		return nil, fmt.Errorf("Found wrong number of paths to codeql-database.yml: %s", paths)
	}

	data, err := os.ReadFile(paths[0])
	if err != nil {
		awaitSignal("XX: getDataBaseMetadata ", "databasePath=", databasePath)
		return nil, fmt.Errorf("failed to read database metadata: %v", err)
	}

	var metadata DatabaseMetadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal database metadata: %v", err)
	}

	return &metadata, nil
}

func runCommand(command []string) (CodeQLCommandOutput, error) {
	slog.Info("Running command", "command", command)
	cmd := exec.Command(command[0], command[1:]...)
	stdout, err := cmd.Output()
	if err != nil {
		return CodeQLCommandOutput{ExitCode: 1}, err
	}
	return CodeQLCommandOutput{ExitCode: 0, Stdout: string(stdout)}, nil
}

func validateQueryMetadataObject(data []byte) (QueryMetadata, error) {
	var queryMetadata QueryMetadata
	if err := json.Unmarshal(data, &queryMetadata); err != nil {
		return QueryMetadata{}, err
	}
	return queryMetadata, nil
}

func validateBQRSInfoObject(data []byte) (BQRSInfo, error) {
	var bqrsInfo BQRSInfo
	if err := json.Unmarshal(data, &bqrsInfo); err != nil {
		return BQRSInfo{}, err
	}
	return bqrsInfo, nil
}

func getBqrsInfo(codeql CodeqlCli, bqrs string) (BQRSInfo, error) {
	bqrsInfoOutput, err := runCommand([]string{codeql.Path, "bqrs", "info", "--format=json", bqrs})
	if err != nil {
		return BQRSInfo{}, fmt.Errorf("unable to run codeql bqrs info. Error: %v", err)
	}
	if bqrsInfoOutput.ExitCode != 0 {
		return BQRSInfo{}, fmt.Errorf("unable to run codeql bqrs info. Exit code: %d", bqrsInfoOutput.ExitCode)
	}
	return validateBQRSInfoObject([]byte(bqrsInfoOutput.Stdout))
}

func getQueryMetadata(codeql CodeqlCli, query string) (QueryMetadata, error) {
	queryMetadataOutput, err := runCommand([]string{codeql.Path, "resolve", "metadata", "--format=json", query})
	if err != nil {
		return QueryMetadata{}, fmt.Errorf("unable to run codeql resolve metadata. Error: %v", err)
	}
	if queryMetadataOutput.ExitCode != 0 {
		return QueryMetadata{}, fmt.Errorf("unable to run codeql resolve metadata. Exit code: %d", queryMetadataOutput.ExitCode)
	}
	return validateQueryMetadataObject([]byte(queryMetadataOutput.Stdout))
}

func getQueryPackRunResults(codeql CodeqlCli, databasePath, queryPackPath string) (*QueryPackRunResults, error) {
	resultsBasePath := filepath.Join(databasePath, "results")

	queryPaths := []string{} // Replace with actual query paths resolution logic

	var queries []Query
	for _, queryPath := range queryPaths {
		relativeBqrsFilePath := filepath.Join(queryPackPath, queryPath)
		bqrsFilePath := filepath.Join(resultsBasePath, relativeBqrsFilePath)

		if _, err := os.Stat(bqrsFilePath); os.IsNotExist(err) {
			return nil, fmt.Errorf("could not find BQRS file for query %s at %s", queryPath, bqrsFilePath)
		}

		bqrsInfo, err := getBqrsInfo(codeql, bqrsFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get BQRS info: %v", err)
		}

		queryMetadata, err := getQueryMetadata(codeql, queryPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get query metadata: %v", err)
		}

		queries = append(queries, Query{
			QueryPath:            queryPath,
			QueryMetadata:        queryMetadata,
			RelativeBqrsFilePath: relativeBqrsFilePath,
			BqrsInfo:             bqrsInfo,
		})
	}

	totalResultsCount := 0
	for _, query := range queries {
		count, err := getBqrsResultCount(query.BqrsInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to get BQRS result count: %v", err)
		}
		totalResultsCount += count
	}

	return &QueryPackRunResults{
		Queries:           queries,
		TotalResultsCount: totalResultsCount,
		ResultsBasePath:   resultsBasePath,
	}, nil
}

func adjustBqrsFiles(queryPackRunResults *QueryPackRunResults, resultsDir string) (BqrsFilePaths, error) {
	if len(queryPackRunResults.Queries) == 1 {
		currentBqrsFilePath := filepath.Join(queryPackRunResults.ResultsBasePath, queryPackRunResults.Queries[0].RelativeBqrsFilePath)
		newBqrsFilePath := filepath.Join(resultsDir, "results.bqrs")

		if err := os.MkdirAll(resultsDir, os.ModePerm); err != nil {
			return BqrsFilePaths{}, err
		}

		if err := os.Rename(currentBqrsFilePath, newBqrsFilePath); err != nil {
			return BqrsFilePaths{}, err
		}

		return BqrsFilePaths{BasePath: resultsDir, RelativeFilePaths: []string{"results.bqrs"}}, nil
	}

	relativeFilePaths := make([]string, len(queryPackRunResults.Queries))
	for i, query := range queryPackRunResults.Queries {
		relativeFilePaths[i] = query.RelativeBqrsFilePath
	}

	return BqrsFilePaths{
		BasePath:          queryPackRunResults.ResultsBasePath,
		RelativeFilePaths: relativeFilePaths,
	}, nil
}

func getSourceLocationPrefix(codeql CodeqlCli, databasePath string) (string, error) {
	cmd := exec.Command(codeql.Path, "resolve", "database", databasePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("Failed to resolve database", "err", err, "output", output)
		return "", fmt.Errorf("failed to resolve database: %v\nOutput: %s", err, output)
	}

	var resolvedDatabase ResolvedDatabase
	if err := json.Unmarshal(output, &resolvedDatabase); err != nil {
		return "", fmt.Errorf("failed to unmarshal resolved database: %v", err)
	}

	return resolvedDatabase.SourceLocationPrefix, nil
}

func queryPackSupportsSarif(queryPackRunResults *QueryPackRunResults) bool {
	for _, query := range queryPackRunResults.Queries {
		if !querySupportsSarif(query.QueryMetadata, query.BqrsInfo) {
			return false
		}
	}
	return true
}

func querySupportsSarif(queryMetadata QueryMetadata, bqrsInfo BQRSInfo) bool {
	return getSarifOutputType(queryMetadata, bqrsInfo.CompatibleQueryKinds) != ""
}

func getSarifOutputType(queryMetadata QueryMetadata, compatibleQueryKinds []string) string {
	if (*queryMetadata.Kind == "path-problem" || *queryMetadata.Kind == "path-alert") && contains(compatibleQueryKinds, "PathProblem") {
		return "path-problem"
	}
	if (*queryMetadata.Kind == "problem" || *queryMetadata.Kind == "alert") && contains(compatibleQueryKinds, "Problem") {
		return "problem"
	}
	return ""
}

func generateSarif(codeql CodeqlCli, language queue.QueryLanguage, databasePath, queryPackPath, databaseSHA string, resultsDir string) ([]byte, error) {
	sarifFile := filepath.Join(resultsDir, "results.sarif")
	cmd := exec.Command(codeql.Path, "database", "interpret-results", "--format=sarif-latest", "--output="+sarifFile, "--sarif-add-snippets", "--no-group-results", databasePath, queryPackPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to generate SARIF: %v\nOutput: %s", err, output)
	}

	sarifData, err := os.ReadFile(sarifFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read SARIF file: %v", err)
	}

	// Modify the sarif: start by extracting
	var sarif Sarif
	if err := json.Unmarshal(sarifData, &sarif); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SARIF: %v", err)
	}

	// now inject version control info
	for _, run := range sarif.Runs {
		run.VersionControlProvenance = append(run.VersionControlProvenance, map[string]interface{}{
			"repositoryUri": fmt.Sprintf("%s/%s", os.Getenv("GITHUB_SERVER_URL"), language),
			"revisionId":    databaseSHA,
		})
	}

	// and write it back
	sarifBytes, err := json.Marshal(sarif)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SARIF: %v", err)
	}

	return sarifBytes, nil
}

// XX: inlined this function
// func injectVersionControlInfo(sarif *Sarif, nwo, databaseSHA string) {
// 	// XX: is nwo name/owner or language?
// 	slog.Debug("XX: 2: is nwo a name/owner, or the original callers' queryLanguage?",
// 		"nwo", nwo)
// 	for _, run := range sarif.Runs {
// 		run.VersionControlProvenance = append(run.VersionControlProvenance, map[string]interface{}{
// 			"repositoryUri": fmt.Sprintf("%s/%s", os.Getenv("GITHUB_SERVER_URL"), nwo),
// 			"revisionId":    databaseSHA,
// 		})
// 	}
// }

// getSarifResultCount returns the number of results in the SARIF file.
func getSarifResultCount(sarif []byte) int {
	var sarifData Sarif
	if err := json.Unmarshal(sarif, &sarifData); err != nil {
		log.Printf("failed to unmarshal SARIF for result count: %v", err)
		return 0
	}
	count := 0
	for _, run := range sarifData.Runs {
		count += len(run.Results)
	}
	return count
}

// Known result set names
var KnownResultSetNames = []string{"#select", "problems"}

// getBqrssResultCount returns the number of results in the BQRS file.
func getBqrsResultCount(bqrsInfo BQRSInfo) (int, error) {
	for _, name := range KnownResultSetNames {
		for _, resultSet := range bqrsInfo.ResultSets {
			if resultSet.Name == name {
				return resultSet.Rows, nil
			}
		}
	}
	var resultSetNames []string
	for _, resultSet := range bqrsInfo.ResultSets {
		resultSetNames = append(resultSetNames, resultSet.Name)
	}
	return 0, fmt.Errorf(
		"BQRS does not contain any result sets matching known names. Expected one of %s but found %s",
		KnownResultSetNames, resultSetNames,
	)
}

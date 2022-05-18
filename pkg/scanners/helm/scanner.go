package helm

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/liamg/memoryfs"

	"github.com/aquasecurity/defsec/internal/debug"
	"github.com/aquasecurity/defsec/internal/types"
	"github.com/aquasecurity/defsec/pkg/scan"
	"github.com/aquasecurity/defsec/pkg/scanners/helm/parser"
	kparser "github.com/aquasecurity/defsec/pkg/scanners/kubernetes/parser"
	"github.com/aquasecurity/defsec/pkg/scanners/options"

	"github.com/aquasecurity/defsec/pkg/rego"
)

type Scanner struct {
	policyDirs    []string
	dataDirs      []string
	debug         debug.Logger
	options       []options.ScannerOption
	policyReaders []io.Reader
	loadEmbedded  bool
	policyFS      fs.FS
	skipRequired  bool
}

// New creates a new Scanner
func New(options ...options.ScannerOption) *Scanner {
	s := &Scanner{
		options: options,
	}

	for _, option := range options {
		option(s)
	}
	return s
}

func (s *Scanner) SetUseEmbeddedPolicies(b bool) {
	s.loadEmbedded = b
}

func (s *Scanner) Name() string {
	return "Helm"
}

func (s *Scanner) SetPolicyReaders(readers []io.Reader) {
	s.policyReaders = readers
}

func (s *Scanner) SetSkipRequiredCheck(skip bool) {
	s.skipRequired = skip
}

func (s *Scanner) SetDebugWriter(writer io.Writer) {
	s.debug = debug.New(writer, "scan:helm")
}

func (s *Scanner) SetTraceWriter(_ io.Writer) {
	// handled by rego later - nothing to do for now...
}

func (s *Scanner) SetPerResultTracingEnabled(_ bool) {
	// handled by rego later - nothing to do for now...
}

func (s *Scanner) SetPolicyDirs(dirs ...string) {
	s.policyDirs = dirs
}

func (s *Scanner) SetDataDirs(dirs ...string) {
	s.dataDirs = dirs
}

func (s *Scanner) SetPolicyNamespaces(namespaces ...string) {
	// handled by rego later - nothing to do for now...
}

func (s *Scanner) SetPolicyFilesystem(policyFS fs.FS) {
	s.policyFS = policyFS
}

func (s *Scanner) ScanFS(ctx context.Context, target fs.FS, path string) (scan.Results, error) {

	helmParser := parser.New()

	if err := helmParser.ParseFS(ctx, target, path); err != nil {
		return nil, err
	}

	chartFiles, err := helmParser.RenderedChartFiles()
	if err != nil { // not valid helm, maybe some other yaml etc., abort
		return nil, nil
	}

	var results []scan.Result
	regoScanner := rego.NewScanner(s.options...)
	s.loadEmbedded = len(s.policyDirs)+len(s.policyReaders) == 0
	policyFS := target
	if s.policyFS != nil {
		policyFS = s.policyFS
	}
	if err := regoScanner.LoadPolicies(s.loadEmbedded, policyFS, s.policyDirs, s.policyReaders); err != nil {
		return nil, fmt.Errorf("policies load: %w", err)
	}
	for _, file := range chartFiles {
		s.debug.Log("Processing rendered chart file: %s", file.TemplateFilePath)

		manifests, err := kparser.New().Parse(strings.NewReader(file.ManifestContent), file.TemplateFilePath)
		if err != nil {
			return nil, fmt.Errorf("unmarshal yaml: %w", err)
		}
		for _, manifest := range manifests {
			fileResults, err := regoScanner.ScanInput(context.Background(), rego.Input{
				Path:     file.TemplateFilePath,
				Contents: manifest,
				Type:     types.SourceKubernetes,
			})
			if err != nil {
				return nil, fmt.Errorf("scanning error: %w", err)
			}

			if len(fileResults) > 0 {
				renderedFS := memoryfs.New()
				if err := renderedFS.MkdirAll(filepath.Dir(file.TemplateFilePath), fs.ModePerm); err != nil {
					return nil, err
				}
				if err := renderedFS.WriteLazyFile(file.TemplateFilePath, func() (io.Reader, error) {
					return strings.NewReader(file.ManifestContent), nil
				}, fs.ModePerm); err != nil {
					return nil, err
				}
				fileResults.SetSourceAndFilesystem("", renderedFS)
			}

			results = append(results, fileResults...)
		}
	}

	return results, nil

}
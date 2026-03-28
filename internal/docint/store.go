package docint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sambeau/kanbanzai/internal/fsutil"

	"gopkg.in/yaml.v3"
)

// IndexStore handles reading and writing document intelligence index files.
// Files live in .kbz/index/documents/ (per-document) and .kbz/index/ (graph, concepts).
type IndexStore struct {
	indexRoot string
}

// NewIndexStore creates an IndexStore rooted at the given directory.
func NewIndexStore(indexRoot string) *IndexStore {
	return &IndexStore{indexRoot: indexRoot}
}

// SaveDocumentIndex saves a per-document index file.
func (s *IndexStore) SaveDocumentIndex(index *DocumentIndex) error {
	dir := filepath.Join(s.indexRoot, "documents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create index directory: %w", err)
	}
	path := filepath.Join(dir, indexFileName(index.DocumentID))
	return writeYAMLFile(path, index)
}

// LoadDocumentIndex loads a per-document index file.
func (s *IndexStore) LoadDocumentIndex(docID string) (*DocumentIndex, error) {
	path := filepath.Join(s.indexRoot, "documents", indexFileName(docID))
	var index DocumentIndex
	if err := readYAMLFile(path, &index); err != nil {
		return nil, err
	}
	return &index, nil
}

// ListDocumentIndexes returns all indexed document IDs.
func (s *IndexStore) ListDocumentIndexes() ([]string, error) {
	dir := filepath.Join(s.indexRoot, "documents")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".yaml")
		id = strings.ReplaceAll(id, "--", "/")
		ids = append(ids, id)
	}
	return ids, nil
}

// SaveGraph saves the document graph.
func (s *IndexStore) SaveGraph(graph *DocumentGraph) error {
	path := filepath.Join(s.indexRoot, "graph.yaml")
	return writeYAMLFile(path, graph)
}

// LoadGraph loads the document graph. Returns an empty graph if the file does not exist.
func (s *IndexStore) LoadGraph() (*DocumentGraph, error) {
	path := filepath.Join(s.indexRoot, "graph.yaml")
	var graph DocumentGraph
	if err := readYAMLFile(path, &graph); err != nil {
		if os.IsNotExist(err) {
			return &DocumentGraph{}, nil
		}
		return nil, err
	}
	return &graph, nil
}

// SaveConceptRegistry saves the concept registry.
func (s *IndexStore) SaveConceptRegistry(registry *ConceptRegistry) error {
	path := filepath.Join(s.indexRoot, "concepts.yaml")
	return writeYAMLFile(path, registry)
}

// LoadConceptRegistry loads the concept registry. Returns an empty registry if the file does not exist.
func (s *IndexStore) LoadConceptRegistry() (*ConceptRegistry, error) {
	path := filepath.Join(s.indexRoot, "concepts.yaml")
	var registry ConceptRegistry
	if err := readYAMLFile(path, &registry); err != nil {
		if os.IsNotExist(err) {
			return &ConceptRegistry{}, nil
		}
		return nil, err
	}
	return &registry, nil
}

// DocumentIndexExists checks if an index exists for a document.
func (s *IndexStore) DocumentIndexExists(docID string) bool {
	path := filepath.Join(s.indexRoot, "documents", indexFileName(docID))
	_, err := os.Stat(path)
	return err == nil
}

func indexFileName(docID string) string {
	safe := strings.ReplaceAll(docID, "/", "--")
	return safe + ".yaml"
}

func writeYAMLFile(path string, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal YAML: %w", err)
	}

	// Ensure trailing newline
	if len(data) == 0 || data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}

	// Create directory
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Use atomic write to prevent corruption
	return fsutil.WriteFileAtomic(path, data, 0o644)
}

func readYAMLFile(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, v)
}

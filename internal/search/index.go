// Package search provides full-text search functionality for beans using Bleve.
package search

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/xRiErOS/beans/pkg/bean"
)

// Index wraps a Bleve index for searching beans. It may be a private
// in-memory index (persistent == false) or one backed by a persisted,
// locked on-disk index shared across invocations of this store
// (persistent == true); see Open.
type Index struct {
	index      bleve.Index
	persistent bool
	dir        string            // index directory; empty for in-memory indexes
	etags      map[string]string // bean ID -> last-synced ETag; nil for in-memory indexes
	lock       *indexLock        // held only when persistent
}

// beanDocument is the structure stored in the Bleve index.
type beanDocument struct {
	ID    string `json:"id"`
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

// NewIndex creates a new in-memory Bleve index.
func NewIndex() (*Index, error) {
	indexMapping := buildIndexMapping()
	idx, err := bleve.NewMemOnly(indexMapping)
	if err != nil {
		return nil, err
	}

	return &Index{index: idx, etags: map[string]string{}}, nil
}

// buildIndexMapping creates the Bleve index mapping for bean documents.
func buildIndexMapping() mapping.IndexMapping {
	// Create a text field mapping with the standard analyzer
	textFieldMapping := bleve.NewTextFieldMapping()
	textFieldMapping.Analyzer = "standard"

	// Create a keyword field mapping for ID (stored but not analyzed)
	keywordFieldMapping := bleve.NewKeywordFieldMapping()

	// Create the document mapping
	beanMapping := bleve.NewDocumentMapping()
	beanMapping.AddFieldMappingsAt("id", keywordFieldMapping)
	beanMapping.AddFieldMappingsAt("slug", textFieldMapping)
	beanMapping.AddFieldMappingsAt("title", textFieldMapping)
	beanMapping.AddFieldMappingsAt("body", textFieldMapping)

	// Create the index mapping with BM25 scoring for better relevance ranking
	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultMapping = beanMapping
	indexMapping.DefaultAnalyzer = "standard"
	indexMapping.IndexDynamic = false
	indexMapping.StoreDynamic = false

	// Use BM25 scoring algorithm (available in Bleve v2.5.0+)
	// BM25 provides better relevance ranking than TF-IDF, especially for:
	// - Handling term frequency saturation (repeated terms don't over-boost)
	// - Normalizing for document length (short docs aren't unfairly penalized)
	indexMapping.ScoringModel = "bm25"

	return indexMapping
}

// Close closes the index and, for a persistent index, releases the
// cross-process lock acquired by Open.
func (idx *Index) Close() error {
	err := idx.index.Close()
	idx.lock.release()
	return err
}

// staleKey is the freshness signal Sync/IndexBean record and compare: the
// bean's ETag alone is blind to a rename or move, because Slug/Path are
// excluded from the rendered front matter (`yaml:"-"`) and therefore never
// affect ETag. Folding Path into the key means a `beans rename` or a
// `git checkout` that only changes the filename is still detected as a
// change worth reindexing, even though the file's bytes (and thus the ETag
// component) did not move.
func staleKey(b *bean.Bean) string {
	return b.ETag() + "\x1f" + b.Path
}

// IndexBean adds or updates a bean in the search index. For a persistent
// index this also records the bean's current staleness key in the sidecar,
// so a later Sync in another process recognizes this bean as already
// current.
func (idx *Index) IndexBean(b *bean.Bean) error {
	doc := beanDocument{
		ID:    b.ID,
		Slug:  b.Slug,
		Title: b.Title,
		Body:  b.Body,
	}
	if err := idx.index.Index(b.ID, doc); err != nil {
		return err
	}
	idx.etags[b.ID] = staleKey(b)
	if idx.persistent {
		return saveETags(idx.dir, idx.etags)
	}
	return nil
}

// DeleteBean removes a bean from the search index and, for a persistent
// index, its ETag sidecar entry.
func (idx *Index) DeleteBean(id string) error {
	if err := idx.index.Delete(id); err != nil {
		return err
	}
	delete(idx.etags, id)
	if idx.persistent {
		return saveETags(idx.dir, idx.etags)
	}
	return nil
}

// DefaultSearchLimit is the default maximum number of search results.
const DefaultSearchLimit = 1000

// Search executes a search query and returns matching bean IDs.
// The limit parameter controls the maximum number of results (0 uses DefaultSearchLimit).
func (idx *Index) Search(queryStr string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = DefaultSearchLimit
	}

	// Use query string syntax which supports:
	// - Simple terms: "authentication"
	// - Boolean operators: "user AND password"
	// - Wildcards: "auth*"
	// - Phrases: "\"user login\""
	// - Field-specific: "title:login"
	query := bleve.NewQueryStringQuery(queryStr)

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = limit
	searchRequest.Fields = []string{"id"} // Only return ID field

	result, err := idx.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(result.Hits))
	for _, hit := range result.Hits {
		ids = append(ids, hit.ID)
	}

	return ids, nil
}

// Sync brings the index up to date with the given beans, indexing only what
// changed since the last Sync (AC-02): a bean whose staleness key (ETag plus
// Path, see staleKey) differs from the recorded value -- new, edited by this
// process, or edited on disk by another process or an editor since this
// index was last synced -- is reindexed; a bean no longer present is
// removed. ETag, not mtime, is the staleness signal (AC-03); Path is folded
// in because a rename alone never changes ETag (AC-02).
//
// The diff runs the same way for a persistent and an in-memory index: only
// the persisted sidecar write at the end is conditional. An in-memory index
// starts with an empty idx.etags (see NewIndex), so its first Sync in a
// fresh process still reindexes everything; within one process, a later
// Sync on the same Index is incremental exactly like the persistent case,
// including retracting beans no longer present -- swapping the underlying
// Bleve index instead would race Core.Search, which reads idx.index after
// releasing its lock.
func (idx *Index) Sync(beans []*bean.Bean) error {
	seen := make(map[string]struct{}, len(beans))
	batch := idx.index.NewBatch()
	dirty := false
	for _, b := range beans {
		seen[b.ID] = struct{}{}
		key := staleKey(b)
		if idx.etags[b.ID] == key {
			continue
		}
		doc := beanDocument{ID: b.ID, Slug: b.Slug, Title: b.Title, Body: b.Body}
		if err := batch.Index(b.ID, doc); err != nil {
			return err
		}
		idx.etags[b.ID] = key
		dirty = true
	}
	for id := range idx.etags {
		if _, ok := seen[id]; ok {
			continue
		}
		batch.Delete(id)
		delete(idx.etags, id)
		dirty = true
	}
	if !dirty {
		return nil
	}
	if batch.Size() > 0 {
		if err := idx.index.Batch(batch); err != nil {
			return err
		}
	}
	if idx.persistent {
		return saveETags(idx.dir, idx.etags)
	}
	return nil
}

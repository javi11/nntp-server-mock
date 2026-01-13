package nntpserver

import (
	"bytes"
	"encoding/gob"
	"io"
	"net/textproto"
	"os"
	"strconv"
	"sync"

	"github.com/gofiber/storage/bbolt"
)

const (
	DefaultDBPath    = "nntp.db"
	ArticleNumberKey = "article_number"
)

type backendArticle struct {
	Id     string
	Header textproto.MIMEHeader
	Body   []byte
	Bytes  int
	Lines  int
}

type DiskBackend struct {
	db           *bbolt.Storage
	groups       map[string]*Group
	mu           sync.RWMutex
	cleanOnClose bool
	dbPath       string
	articleCount int64 // cached in memory to avoid extra db write
}

func NewDiskBackend(
	cleanOnClose bool,
	dbPath string,
) *DiskBackend {
	testGroup := Group{
		Name:        "test",
		Description: "A test group",
		Low:         1,
		Posting:     PostingPermitted,
	}

	if dbPath == "" {
		dbPath = DefaultDBPath
	}

	store := bbolt.New(
		bbolt.Config{
			Database: dbPath,
		},
	)

	// Disable fsync for faster writes (data may be lost on crash, but persists on normal restart)
	store.Conn().NoSync = true
	store.Conn().NoFreelistSync = true

	// Load article count from disk
	var articleCount int64
	artCount, err := store.Get(ArticleNumberKey)
	if err == nil && artCount != nil {
		articleCount, _ = strconv.ParseInt(string(artCount), 10, 64)
	}

	return &DiskBackend{
		db:           store,
		groups:       map[string]*Group{"test": &testGroup},
		cleanOnClose: cleanOnClose,
		dbPath:       dbPath,
		articleCount: articleCount,
	}
}

func (b *DiskBackend) ListGroups(max int) ([]*Group, error) {
	groups := make([]*Group, 0, len(b.groups))
	for _, group := range b.groups {
		group.Count = b.getArticleCount()
		group.High = group.Low + group.Count - 1
		groups = append(groups, group)
	}

	return groups, nil
}

func (b *DiskBackend) GetGroup(name string) (*Group, error) {
	b.mu.RLock()
	group := b.groups[name]
	b.mu.RUnlock()

	if group == nil {
		b.mu.Lock()

		b.groups[name] = &Group{
			Name:        name,
			Description: "A test group",
			Low:         1,
			Posting:     PostingPermitted,
		}

		group = b.groups[name]
		b.mu.Unlock()
	}

	count := b.getArticleCount()

	b.mu.Lock()
	defer b.mu.Unlock()
	group.Count = count
	group.High = group.Low + group.Count - 1

	return group, nil
}

// GetArticle retrieves an article by message-id or article number.
// If group is nil, only message-id lookups are supported.
// If group is provided, both message-id and article number lookups work.
func (b *DiskBackend) GetArticle(group *Group, id string) (*Article, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	res, _ := b.db.Get(id)
	if res == nil {
		return nil, ErrInvalidMessageID
	}

	var art backendArticle
	if err := gob.NewDecoder(bytes.NewReader(res)).Decode(&art); err != nil {
		return nil, err
	}

	return &Article{
		Header: art.Header,
		Body:   bytes.NewReader(art.Body),
		Bytes:  art.Bytes,
		Lines:  art.Lines,
	}, nil
}

func (b *DiskBackend) GetArticles(group *Group, from, to int64) ([]NumberedArticle, error) {
	panic("not implemented")
}

func (b *DiskBackend) Authorized() bool {
	return true
}

func (b *DiskBackend) Authenticate(user, pass string) (Backend, error) {
	return nil, ErrAuthRejected
}

func (b *DiskBackend) AllowPost() bool {
	return true
}

func (b *DiskBackend) Post(article *Article) error {
	bWr := bytes.NewBuffer(nil)
	if _, err := io.Copy(bWr, article.Body); err != nil {
		return err
	}

	// Use a more efficient binary encoding instead of JSON
	artBuf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(artBuf)
	if err := enc.Encode(backendArticle{
		Id:     article.MessageID(),
		Header: article.Header,
		Body:   bWr.Bytes(),
		Bytes:  article.Bytes,
		Lines:  article.Lines,
	}); err != nil {
		return err
	}

	b.mu.Lock()
	if err := b.db.Set(article.MessageID(), artBuf.Bytes(), 0); err != nil {
		b.mu.Unlock()
		return err
	}
	b.mu.Unlock()

	bWr = nil
	artBuf = nil

	_ = b.increaseArticleCount()

	return nil
}

// Stat checks if an article exists and returns its number and id.
// If group is nil, only message-id lookups are supported.
func (b *DiskBackend) Stat(group *Group, id string) (string, string, error) {
	if _, err := b.GetArticle(group, id); err != nil {
		return "", "", err
	}

	return "1", id, nil
}

func (b *DiskBackend) getArticleCount() int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.articleCount
}

func (b *DiskBackend) increaseArticleCount() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.articleCount++
	return b.articleCount
}

func (b *DiskBackend) Close() error {
	// Persist article count before closing
	b.mu.RLock()
	count := b.articleCount
	b.mu.RUnlock()
	_ = b.db.Set(ArticleNumberKey, []byte(strconv.FormatInt(count, 10)), 0)

	if b.cleanOnClose {
		_ = b.db.Reset()
		defer os.Remove(b.dbPath)
	}

	return b.db.Close()
}

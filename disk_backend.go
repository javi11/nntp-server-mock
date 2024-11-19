package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/textproto"
	"strconv"
	"sync"

	"github.com/gofiber/storage/bbolt"
)

const ArticleNumberKey = "article_number"

type backendArticle struct {
	Id     string
	Header textproto.MIMEHeader
	Body   []byte
	Bytes  int
	Lines  int
}

type DiskBackend struct {
	db     *bbolt.Storage
	mu     sync.Mutex
	groups map[string]*Group
}

func NewDiskBackend() *DiskBackend {
	testGroup := Group{
		Name:        "test",
		Description: "A test group",
		Low:         1,
		Posting:     PostingPermitted,
	}

	store := bbolt.New()

	return &DiskBackend{
		db:     store,
		groups: map[string]*Group{"test": &testGroup},
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
	group := b.groups[name]
	if group == nil {
		b.groups[name] = &Group{
			Name:        name,
			Description: "A test group",
			Low:         1,
			Posting:     PostingPermitted,
		}

		group = b.groups[name]
	}

	group.Count = b.getArticleCount()
	group.High = group.Low + group.Count - 1

	return group, nil
}

func (b *DiskBackend) GetArticle(group *Group, id string) (*Article, error) {
	b.mu.Lock()
	res, err := b.db.Get(id)
	b.mu.Unlock()

	if err != nil {
		return nil, ErrInvalidMessageID
	}

	var art backendArticle
	if err = json.NewDecoder(bytes.NewReader(res)).Decode(&art); err != nil {
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

	artBuf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(artBuf).Encode(backendArticle{
		Id:     article.MessageID(),
		Header: article.Header,
		Body:   bWr.Bytes(),
		Bytes:  article.Bytes,
		Lines:  article.Lines,
	}); err != nil {
		return err
	}

	if err := b.db.Set(article.MessageID(), artBuf.Bytes(), 0); err != nil {
		return err
	}

	_ = b.increaseArticleCount()

	return nil
}

func (b *DiskBackend) Stat(group *Group, id string) (string, string, error) {
	if _, err := b.GetArticle(group, id); err != nil {
		return "", "", err
	}

	return "1", id, nil
}

func (b *DiskBackend) getArticleCount() int64 {
	artCount, err := b.db.Get(ArticleNumberKey)
	if err != nil {
		artCount = []byte("0")
	}

	count, _ := strconv.ParseInt(string(artCount), 10, 64)
	return count
}

func (b *DiskBackend) increaseArticleCount() int64 {
	artCount, err := b.db.Get(ArticleNumberKey)
	if err != nil {
		artCount = []byte("0")
	}

	count, _ := strconv.ParseInt(string(artCount), 10, 64)

	count++

	if err := b.db.Set(ArticleNumberKey, []byte(strconv.FormatInt(count, 10)), 0); err != nil {
		return 0
	}

	return count
}

func (b *DiskBackend) Close() error {
	return b.db.Close()
}

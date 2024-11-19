package main

import (
	"fmt"
	"io"
	"net/textproto"
)

// PostingStatus type for groups.
type PostingStatus byte

// PostingStatus values.
const (
	Unknown             = PostingStatus(0)
	PostingPermitted    = PostingStatus('y')
	PostingNotPermitted = PostingStatus('n')
	PostingModerated    = PostingStatus('m')
)

func (ps PostingStatus) String() string {
	return fmt.Sprintf("%c", ps)
}

type Group struct {
	Name        string
	Description string
	Count       int64
	High        int64
	Low         int64
	Posting     PostingStatus
}

type Article struct {
	Header textproto.MIMEHeader
	Body   io.Reader
	Bytes  int
	Lines  int
}

func (a *Article) MessageID() string {
	return a.Header.Get("Message-Id")
}

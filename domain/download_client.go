package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type DownloadClientType string

const (
	DownloadClientQBittorrent DownloadClientType = "qbittorrent"
	DownloadClientAuthorToday DownloadClientType = "author_today"
)

var validDownloadClientTypes = map[DownloadClientType]bool{
	DownloadClientQBittorrent:   true,
	DownloadClientAuthorToday:   true,
}

type DownloadClient struct {
	Id       ID
	Name     string
	Type     DownloadClientType
	Settings map[string]string
	Enabled  bool
}

func (c *DownloadClient) Get(key string) string {
	if c == nil || c.Settings == nil {
		return ""
	}
	return c.Settings[key]
}

func (c *DownloadClient) ApplyDefaultName() {
	if c == nil {
		return
	}
	c.Name = strings.TrimSpace(c.Name)
	if c.Name != "" {
		return
	}
	switch c.Type {
	case DownloadClientQBittorrent:
		c.Name = "qBittorrent"
	case DownloadClientAuthorToday:
		c.Name = "Author.today"
	default:
		c.Name = string(c.Type)
	}
}

func (c *DownloadClient) Validate() error {
	c.ApplyDefaultName()
	if c.Name == "" {
		return ErrEmptyName
	}
	if !validDownloadClientTypes[c.Type] {
		return fmt.Errorf("%w: %s", ErrInvalidDownloadClientType, c.Type)
	}
	switch c.Type {
	case DownloadClientQBittorrent:
		if c.Get("host") == "" {
			return errors.New("download client host is required")
		}
	case DownloadClientAuthorToday:
		if c.Get("token") == "" {
			return errors.New("author.today token is required")
		}
	}
	return nil
}

type DownloadClientRepository interface {
	Add(c *DownloadClient) error
	GetById(id ID) (*DownloadClient, error)
	List() ([]DownloadClient, error)
	Update(c *DownloadClient) error
	Remove(id ID) error
}

type TestableClient interface {
	TestConnection(ctx context.Context) error
}

type DownloadClientTester interface {
	Test(ctx context.Context, c DownloadClient) error
}

package importer

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"mut/internal/catalog"
	"mut/internal/vault"
	"path/filepath"
	"strings"
	"uuid"
)

type fb2book struct {
	XMLName xml.Name    `xml:"FictionBook"`
	Title   string      `xml:"description>title-info>book-title"`
	Author  []fb2Author `xml:"description>title-info>author"`
}

type fb2Author struct {
	FirstName  string `xml:"first-name"`
	MiddleName string `xml:"middle-name"`
	LastName   string `xml:"last-name"`
}

type epubContainer struct {
	RootFile struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type opfPackage struct {
	Title    string   `xml:"metadata>title"`
	Creators []string `xml:"metadata>creator"`
}

var ErrDuplicateBook = errors.New("book already exists")

func ImportFile(filename string, v *vault.Vault, c *catalog.Catalog, data []byte, keepDuplicate bool) error {
	sum := sha256.Sum256(data)
	sha256 := hex.EncodeToString(sum[:])
	book, _, err := c.FindByHash(sha256)
	if err != nil {
		return fmt.Errorf("find by hash error: %v", err)
	}
	if book.ID != uuid.Nil() && !keepDuplicate {
		return ErrDuplicateBook
	}
	objectID, err := v.PutObject(data)
	if err != nil {
		return fmt.Errorf("cannot save object error: %w", err)
	}
	ext := filepath.Ext(filename)
	book.FileObjectID = objectID
	book.Sha256 = sha256
	book.Originalname = filename
	book.FileSize = len(data)
	book.ID = uuid.New()
	book.Format = strings.ToUpper(strings.TrimLeft(ext, "."))
	if book.Format == "FB2" {
		title, authorname, err := ParseFB2(data)
		if err != nil {
			log.Printf("cannot parse fb2 error: %v", err)
		}
		if book.Title == "" {
			book.Title = title
		}
		if book.Author == "" {
			book.Author = authorname
		}
	} else if book.Format == "EPUB" {
		title, authorname, err := ParseEPUB(data)
		if err != nil {
			log.Printf("cannot parse epub error: %v", err)
		}
		if book.Title == "" {
			book.Title = title
		}
		if book.Author == "" {
			book.Author = authorname
		}
	}
	err = c.AddBook(book)
	if err != nil {
		return fmt.Errorf("cannot add book error: %w", err)
	}
	_, err = c.Save()
	if err != nil {
		return fmt.Errorf("cannot save catalog error: %w", err)
	}

	return nil

}

func ParseFB2(data []byte) (title string, author string, err error) {
	var fb2 fb2book
	err = xml.Unmarshal(data, &fb2)
	if err != nil {
		return "", "", fmt.Errorf("cannot unmarshal fb2 error: %w", err)
	}
	var authors []string
	var authorname string
	for _, author := range fb2.Author {
		authorname = ""
		if author.LastName != "" {
			authorname += author.LastName
		}
		if author.FirstName != "" {
			authorname += " " + author.FirstName
		}
		if author.MiddleName != "" {
			authorname += " " + author.MiddleName
		}
		authors = append(authors, authorname)
	}
	title = strings.TrimSpace(fb2.Title)
	if title == "" {
		title = "unknown title"
	}
	if len(authors) == 0 {
		author = "unknown author"
	} else {
		author = strings.TrimSpace(strings.Join(authors, ", "))
	}
	return title, author, nil
}

func ParseEPUB(data []byte) (title string, author string, err error) {
	reader := bytes.NewReader(data)
	zipReader, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		return "", "", fmt.Errorf("cannot read zip/epub error: %v", err)
	}
	for _, file := range zipReader.File {
		if strings.EqualFold(file.Name, "META-INF/container.xml") {
			rc, err := file.Open()
			if err != nil {
				return "", "", fmt.Errorf("cannot open container.xml error: %v", err)
			}
			defer rc.Close()
			content, err := io.ReadAll(io.LimitReader(rc, 2*1024*1024))
			if err != nil {
				return "", "", fmt.Errorf("cannot read first 2mb of epub error: %v", err)
			}
			var container epubContainer
			err = xml.Unmarshal(content, &container)
			if err != nil {
				return "", "", fmt.Errorf("cannot unmarshal container.xml error: %v", err)
			}
			for _, file := range zipReader.File {
				if file.Name == container.RootFile.FullPath {
					rc, err := file.Open()
					if err != nil {
						return "", "", fmt.Errorf("cannot open opf error: %v", err)
					}
					defer rc.Close()
					content, err := io.ReadAll(io.LimitReader(rc, 2*1024*1024))
					if err != nil {
						return "", "", fmt.Errorf("cannot read first 2mb of epub error: %v", err)
					}
					var opf opfPackage
					err = xml.Unmarshal(content, &opf)
					if err != nil {
						return "", "", fmt.Errorf("cannot unmarshal opf error: %v", err)
					}
					title = strings.TrimSpace(opf.Title)
					if title == "" {
						title = "unknown title"
					}
					author = strings.TrimSpace(strings.Join(opf.Creators, ", "))
					if author == "" {
						author = "unknown author"
					}
					return title, author, nil
				}
			}
		}
	}
	return "", "", fmt.Errorf("container.xml or opf not found")
}

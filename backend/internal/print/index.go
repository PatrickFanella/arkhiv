// Package print adapts Arkhiv's bounded PDF indexing at
// 99d499b92cb8d848e0fcb37288ece8bce9de943e (GPL-3.0-only).
package print

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"git.subcult.tv/subculture-collective/vhness/internal/catalog"
	"git.subcult.tv/subculture-collective/vhness/internal/domain"
	"git.subcult.tv/subculture-collective/vhness/internal/preservation"
)

type bounded struct {
	bytes.Buffer
	limit int
}

func (b *bounded) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.limit {
		return 0, fmt.Errorf("PDF tool output exceeds bound")
	}
	return b.Buffer.Write(p)
}
func command(ctx context.Context, input *os.File, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if _, err := input.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "prlimit", append([]string{"--as=536870912", "--cpu=30", "--fsize=33554432", "--", name}, args...)...)
	cmd.ExtraFiles = []*os.File{input}
	output := bounded{limit: 16 << 20}
	stderr := bounded{limit: 64 << 10}
	cmd.Stdout = &output
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s failed or resource limit reached", name)
	}
	return output.Bytes(), nil
}

func original(entry catalog.TransportEntry, file string) (domain.LocalOriginal, error) {
	for _, asset := range entry.Preservation.Manifest {
		if asset.File == file {
			return asset, nil
		}
	}
	return domain.LocalOriginal{}, fmt.Errorf("original outside manifest")
}

func Index(ctx context.Context, db *catalog.DB, mediaRoot, derivativeRoot, item, representation, collection string) error {
	entry, err := db.Transport(ctx, item, representation, collection)
	if err != nil {
		return err
	}
	file := entry.Representation.File
	if entry.Collection.Policy != domain.Local || !strings.HasSuffix(strings.ToLower(file), ".pdf") {
		return fmt.Errorf("indexing requires an already preserved PDF")
	}
	if err = domain.AuthorizeTransport(entry.Collection, entry.Representation, entry.Raw, entry.Review, entry.Preservation, domain.ReadAction, file); err != nil {
		return err
	}
	asset, err := original(entry, file)
	if err != nil {
		return err
	}
	f, err := preservation.Verify(mediaRoot, asset.Path, asset.SHA256, asset.Bytes)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := command(ctx, f, "pdfinfo", "/proc/self/fd/3")
	if err != nil {
		return err
	}
	match := regexp.MustCompile(`(?m)^Pages:\s+(\d+)`).FindSubmatch(info)
	if len(match) != 2 {
		return fmt.Errorf("PDF page count unavailable")
	}
	pages, err := strconv.Atoi(string(match[1]))
	if err != nil || pages < 1 || pages > 10000 {
		return fmt.Errorf("PDF page count exceeds bound")
	}
	text, err := command(ctx, f, "pdftotext", "-layout", "-enc", "UTF-8", "/proc/self/fd/3", "-")
	if err != nil {
		return err
	}
	sections := map[int]string{}
	status := "no-text"
	textFile, textSHA256 := "", ""
	for number, section := range strings.Split(string(text), "\f") {
		if number >= pages {
			break
		}
		if strings.TrimSpace(section) != "" {
			sections[number+1] = section
			status = "page-text"
		}
	}
	if status == "no-text" {
		for _, companion := range entry.Representation.Extension.Print.Companions {
			if companion.Role == "ocr" {
				if err = domain.AuthorizeTransport(entry.Collection, entry.Representation, entry.Raw, entry.Review, entry.Preservation, domain.ReadAction, companion.File); err != nil {
					return err
				}
				ocr, err := original(entry, companion.File)
				if err != nil {
					return err
				}
				if ocr.Bytes > 16<<20 {
					return fmt.Errorf("OCR companion exceeds text bound")
				}
				input, err := preservation.Verify(mediaRoot, ocr.Path, ocr.SHA256, ocr.Bytes)
				if err != nil {
					return err
				}
				data, err := io.ReadAll(io.LimitReader(input, 16<<20+1))
				input.Close()
				if err != nil || len(data) > 16<<20 {
					return fmt.Errorf("OCR companion could not be read within bounds")
				}
				if strings.TrimSpace(string(data)) != "" {
					sections[0] = string(data)
					status = "document-text"
					textFile, textSHA256 = companion.File, ocr.SHA256
				}
				break
			}
		}
	}
	if !filepath.IsAbs(derivativeRoot) {
		return fmt.Errorf("absolute derivative root required")
	}
	if err = os.MkdirAll(derivativeRoot, 0700); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(derivativeRoot, "print-index-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	index := domain.PrintIndex{Snapshot: entry.Representation.Snapshot, OriginalSHA256: asset.SHA256, Pages: pages, TextStatus: status, TextFile: textFile, TextSHA256: textSHA256, Tool: "Poppler pdfinfo/pdftotext/pdftoppm"}
	_, coverErr := command(ctx, f, "pdftoppm", "-f", "1", "-singlefile", "-scale-to", "700", "-png", "/proc/self/fd/3", filepath.Join(staging, "cover"))
	if coverErr == nil {
		cover, err := os.ReadFile(filepath.Join(staging, "cover.png"))
		if err != nil {
			return err
		}
		if len(cover) > 32<<20 {
			return fmt.Errorf("cover exceeds bound")
		}
		hash := domain.Digest(cover)
		path := "covers/" + hash + ".png"
		root, err := os.OpenRoot(derivativeRoot)
		if err != nil {
			return err
		}
		defer root.Close()
		if err = root.MkdirAll("covers", 0700); err != nil {
			return err
		}
		dest, err := root.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0400)
		if err == nil {
			_, err = dest.Write(cover)
			if err == nil {
				err = dest.Sync()
			}
			closeErr := dest.Close()
			if err != nil {
				return err
			}
			if closeErr != nil {
				return closeErr
			}
		} else if !os.IsExist(err) {
			return err
		}
		checked, err := preservation.Verify(derivativeRoot, path, hash, int64(len(cover)))
		if err != nil {
			return err
		}
		checked.Close()
		index.CoverPath = path
		index.CoverSHA256 = hash
		index.CoverBytes = int64(len(cover))
	}
	// Recheck withdrawal/source binding before publishing any derivative index.
	current, err := db.Transport(ctx, item, representation, collection)
	if err != nil {
		return err
	}
	if current.Representation.Snapshot != index.Snapshot {
		return fmt.Errorf("source changed during indexing")
	}
	if err = domain.AuthorizeTransport(current.Collection, current.Representation, current.Raw, current.Review, current.Preservation, domain.ReadAction, file); err != nil {
		return err
	}
	if textFile != "" {
		if err = domain.AuthorizeTransport(current.Collection, current.Representation, current.Raw, current.Review, current.Preservation, domain.ReadAction, textFile); err != nil {
			return err
		}
	}
	return db.StorePrintIndex(ctx, item, representation, file, index, sections)
}

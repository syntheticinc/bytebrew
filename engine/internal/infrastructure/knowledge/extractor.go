package knowledge

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ExtractText converts binary file content to plain text based on file type.
// Supported: txt, md, csv (pass-through), pdf (text extraction), docx (XML parsing).
func ExtractText(content []byte, fileType string) (string, error) {
	switch strings.ToLower(fileType) {
	case "txt", "md", "csv":
		return string(content), nil
	case "pdf":
		return extractPDF(content)
	case "docx":
		return extractDOCX(content)
	default:
		return "", fmt.Errorf("unsupported file type for text extraction: %s", fileType)
	}
}

// extractPDF extracts text content from a PDF file using ledongthuc/pdf.
func extractPDF(content []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("open PDF: %w", err)
	}

	var buf strings.Builder
	numPages := reader.NumPage()
	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue // skip unreadable pages
		}
		if buf.Len() > 0 {
			buf.WriteString("\n\n")
		}
		buf.WriteString(strings.TrimSpace(text))
	}

	result := buf.String()
	if result == "" {
		return "", fmt.Errorf("no text content found in PDF (may be scanned/image-based)")
	}
	return result, nil
}

// extractDOCX extracts text from a DOCX file by parsing the word/document.xml inside the ZIP.
func extractDOCX(content []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("open DOCX: %w", err)
	}

	// Find word/document.xml
	var docFile *zip.File
	for _, f := range reader.File {
		if f.Name == "word/document.xml" {
			docFile = f
			break
		}
	}
	if docFile == nil {
		return "", fmt.Errorf("word/document.xml not found in DOCX archive")
	}

	rc, err := docFile.Open()
	if err != nil {
		return "", fmt.Errorf("open document.xml: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("read document.xml: %w", err)
	}

	return parseDocumentXML(data)
}

// parseDocumentXML extracts text runs from OOXML document.xml.
// Walks <w:t> elements inside <w:p> paragraphs.
func parseDocumentXML(data []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var paragraphs []string
	var currentParagraph strings.Builder
	inText := false

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("parse XML: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			// <w:t> contains actual text runs
			if t.Name.Local == "t" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				inText = true
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inText = false
			}
			// End of paragraph <w:p> → flush
			if t.Name.Local == "p" && t.Name.Space == "http://schemas.openxmlformats.org/wordprocessingml/2006/main" {
				text := strings.TrimSpace(currentParagraph.String())
				if text != "" {
					paragraphs = append(paragraphs, text)
				}
				currentParagraph.Reset()
			}
		case xml.CharData:
			if inText {
				currentParagraph.Write(t)
			}
		}
	}

	// Flush last paragraph
	if text := strings.TrimSpace(currentParagraph.String()); text != "" {
		paragraphs = append(paragraphs, text)
	}

	result := strings.Join(paragraphs, "\n")
	if result == "" {
		return "", fmt.Errorf("no text content found in DOCX")
	}
	return result, nil
}

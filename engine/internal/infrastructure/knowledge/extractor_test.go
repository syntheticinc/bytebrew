package knowledge

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestExtractText_PlainText(t *testing.T) {
	content := []byte("Hello, this is plain text.")
	text, err := ExtractText(content, "txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Hello, this is plain text." {
		t.Errorf("expected plain text pass-through, got %q", text)
	}
}

func TestExtractText_Markdown(t *testing.T) {
	content := []byte("# Header\n\nSome **bold** text.")
	text, err := ExtractText(content, "md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "# Header\n\nSome **bold** text." {
		t.Errorf("expected markdown pass-through, got %q", text)
	}
}

func TestExtractText_CSV(t *testing.T) {
	content := []byte("name,value\ntest,123")
	text, err := ExtractText(content, "csv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "name,value\ntest,123" {
		t.Errorf("expected CSV pass-through, got %q", text)
	}
}

func TestExtractText_UnsupportedType(t *testing.T) {
	_, err := ExtractText([]byte("data"), "xyz")
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestExtractText_InvalidPDF(t *testing.T) {
	_, err := ExtractText([]byte("not a pdf"), "pdf")
	if err == nil {
		t.Fatal("expected error for invalid PDF")
	}
}

func TestExtractText_InvalidDOCX(t *testing.T) {
	_, err := ExtractText([]byte("not a docx"), "docx")
	if err == nil {
		t.Fatal("expected error for invalid DOCX")
	}
}

func TestExtractText_ValidDOCX(t *testing.T) {
	// Build a minimal valid DOCX (ZIP with word/document.xml)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	docXML := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>Hello from DOCX</w:t></w:r></w:p>
    <w:p><w:r><w:t>Second paragraph</w:t></w:r></w:p>
  </w:body>
</w:document>`
	if _, err := w.Write([]byte(docXML)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	text, err := ExtractText(buf.Bytes(), "docx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "Hello from DOCX") {
		t.Errorf("expected 'Hello from DOCX' in output, got %q", text)
	}
	if !strings.Contains(text, "Second paragraph") {
		t.Errorf("expected 'Second paragraph' in output, got %q", text)
	}
}

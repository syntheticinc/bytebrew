package knowledge

import (
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

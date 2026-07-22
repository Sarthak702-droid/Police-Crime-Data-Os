package utils

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
)

// BuildTextPDF creates a small standards-compliant PDF without external
// binaries. Core-font PDFs are intentionally ASCII; unsupported script text is
// replaced in the export until a Unicode font service is configured.
func BuildTextPDF(title string, lines []string) []byte {
	wrapped := []string{title, ""}
	for _, line := range lines {
		wrapped = append(wrapped, wrapPDFLine(line, 92)...)
	}
	const linesPerPage = 54
	pageCount := (len(wrapped) + linesPerPage - 1) / linesPerPage
	if pageCount == 0 {
		pageCount = 1
	}

	objects := make([]string, 3+pageCount*2)
	objects[0] = "<< /Type /Catalog /Pages 2 0 R >>"
	kids := make([]string, 0, pageCount)
	for page := 0; page < pageCount; page++ {
		pageID := 4 + page*2
		contentID := pageID + 1
		kids = append(kids, fmt.Sprintf("%d 0 R", pageID))
		start := page * linesPerPage
		end := start + linesPerPage
		if end > len(wrapped) {
			end = len(wrapped)
		}
		var stream strings.Builder
		stream.WriteString("BT /F1 10 Tf 48 794 Td 13 TL\n")
		for _, line := range wrapped[start:end] {
			fmt.Fprintf(&stream, "(%s) Tj T*\n", escapePDFText(line))
		}
		stream.WriteString("ET")
		objects[pageID-1] = fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>", contentID)
		objects[contentID-1] = fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", stream.Len(), stream.String())
	}
	objects[1] = fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), pageCount)
	objects[2] = "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")
	offsets := make([]int, len(objects)+1)
	for i, object := range objects {
		offsets[i+1] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return out.Bytes()
}

func wrapPDFLine(input string, width int) []string {
	input = strings.Map(func(r rune) rune {
		if r == '\n' {
			return r
		}
		if r < 32 || r > unicode.MaxASCII {
			return '?'
		}
		return r
	}, input)
	var lines []string
	for _, paragraph := range strings.Split(input, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		line := words[0]
		for _, word := range words[1:] {
			if len(line)+1+len(word) > width {
				lines = append(lines, line)
				line = word
			} else {
				line += " " + word
			}
		}
		lines = append(lines, line)
	}
	return lines
}

func escapePDFText(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "(", "\\(")
	return strings.ReplaceAll(value, ")", "\\)")
}

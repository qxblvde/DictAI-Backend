package typst

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Microservices/services/summarization-service/internal/application/interfaces"
)

// escape escapes characters that have special meaning in Typst content.
func escape(s string) string {
	r := strings.NewReplacer(
		`@`, `\@`,
		`#`, `\#`,
		`<`, `\<`,
		`>`, `\>`,
		`$`, `\$`,
		`*`, `\*`,
		`_`, `\_`,
		"`", "\\`",
		`~`, `\~`,
	)
	return r.Replace(s)
}

// preamble contains palette, page setup, typography and the #seg() function
// used by transcript fragments to render per-segment speaker blocks.
const preamble = `
#let bg      = rgb("#FFFFFF")
#let surface = rgb("#F5F5F5")
#let overlay = rgb("#E0E0E0")
#let txt     = rgb("#1A1A1A")
#let heading-txt = rgb("#111111")
#let subtle  = rgb("#888888")

// Speaker colours for light theme — pastel header bg, dark text
#let speaker-colors = (
  (head: rgb("#C8E6C9"), label: rgb("#1B5E20"), border: rgb("#A5D6A7")),  // green
  (head: rgb("#FFCDD2"), label: rgb("#B71C1C"), border: rgb("#EF9A9A")),  // red
  (head: rgb("#FFF9C4"), label: rgb("#F57F17"), border: rgb("#FFF176")),  // yellow
  (head: rgb("#B3E5FC"), label: rgb("#01579B"), border: rgb("#81D4FA")),  // light blue
  (head: rgb("#FCE4EC"), label: rgb("#880E4F"), border: rgb("#F48FB1")),  // pink
  (head: rgb("#C5CAE9"), label: rgb("#1A237E"), border: rgb("#9FA8DA")),  // blue
)

#set page(
  paper: "a4",
  fill: bg,
  margin: (top: 2.8cm, bottom: 2.5cm, left: 2.8cm, right: 2.5cm),
  header: context {
    set text(size: 8pt, fill: subtle, font: "Inter")
    grid(
      columns: (1fr, auto),
      align(left, pageTitle),
      align(right, counter(page).display()),
    )
    v(-0.3em)
    line(length: 100%, stroke: 0.4pt + overlay)
  },
)

#set text(font: "Inter", size: 10.5pt, fill: txt, lang: "ru")
#set par(spacing: 0.85em, leading: 0.62em, justify: true)
#set list(marker: text(fill: subtle)[•], indent: 0.9em, spacing: 0.45em)

#show heading.where(level: 1): it => {
  v(1.8em)
  text(size: 14pt, weight: "bold", fill: heading-txt, font: "Inter", it.body)
  v(0.6em)
}

#let speaker-state = state("ss", (map: (:), count: 0))

#let ensure-speaker(name) = speaker-state.update(s => {
  if name not in s.map {
    s.map.insert(name, calc.rem(s.count, speaker-colors.len()))
    s.count += 1
  }
  s
})

#show heading.where(level: 2): it => {
  let name = it.body.text
  ensure-speaker(name)
  v(0.6em)
  context {
    let s = speaker-state.get()
    let c = speaker-colors.at(s.map.at(name, default: 0))
    block(
      width: 100%,
      radius: 7pt,
      clip: true,
      stroke: 0.5pt + c.border,
    )[
      #block(
        width: 100%,
        fill: c.head,
        inset: (x: 13pt, y: 8pt),
        text(size: 10pt, weight: "bold", fill: c.label, it.body)
      )
    ]
  }
  v(0.2em)
}

#let seg(name, time, body) = {
  ensure-speaker(name)
  context {
    let s = speaker-state.get()
    let c = speaker-colors.at(s.map.at(name, default: 0))
    v(0.45em)
    block(
      width: 100%,
      radius: 7pt,
      clip: true,
      stroke: 0.5pt + c.border,
    )[
      #block(
        width: 100%,
        fill: c.head,
        inset: (x: 13pt, y: 7pt),
      )[
        #set text(size: 9pt, font: "Inter")
        #text(weight: "bold", fill: c.label)[#name]
        #h(1fr)
        #text(fill: c.label.lighten(25%))[#time]
      ]
      #block(
        width: 100%,
        inset: (x: 13pt, y: 10pt),
        text(size: 10pt, fill: txt, body)
      )
    ]
  }
}
`

func buildDoc(title string, meta interfaces.MeetingMeta, contentFragment string) string {
	t := escape(title)
	var sb strings.Builder

	// pageTitle must be defined before #set page() uses it in header
	sb.WriteString("#let pageTitle = [" + t + "]\n")
	sb.WriteString(preamble)

	// ── cover card ────────────────────────────────────────────────────────────
	sb.WriteString("#block(width: 100%, fill: surface, radius: 10pt, inset: (x: 24pt, y: 20pt), stroke: 0.5pt + overlay)[\n")
	sb.WriteString("  #text(size: 27pt, weight: \"bold\", fill: heading-txt, font: \"Inter\")[" + t + "]\n")
	sb.WriteString("  #v(0.55em)\n")
	sb.WriteString("  #line(length: 100%, stroke: 0.5pt + overlay)\n")
	sb.WriteString("  #v(0.45em)\n")

	if meta.WorkspaceName != "" || meta.UploaderName != "" || meta.Date != "" {
		sb.WriteString("  #grid(columns: (auto, 1fr), column-gutter: 1.4em, row-gutter: 0.4em,\n")
		if meta.WorkspaceName != "" {
			sb.WriteString("    text(fill: subtle, size: 8.5pt)[Воркспейс], text(fill: txt, size: 8.5pt)[" + escape(meta.WorkspaceName) + "],\n")
		}
		if meta.UploaderName != "" {
			sb.WriteString("    text(fill: subtle, size: 8.5pt)[Загрузил], text(fill: txt, size: 8.5pt)[" + escape(meta.UploaderName) + "],\n")
		}
		if meta.Date != "" {
			sb.WriteString("    text(fill: subtle, size: 8.5pt)[Дата], text(fill: txt, size: 8.5pt)[" + escape(meta.Date) + "],\n")
		}
		sb.WriteString("  )\n")
	}

	sb.WriteString("]\n")
	sb.WriteString("#v(1.4em)\n\n")

	sb.WriteString(contentFragment)
	sb.WriteString("\n")
	return sb.String()
}

// SummaryDoc builds the full Typst source for a summary document.
func SummaryDoc(meta interfaces.MeetingMeta, contentFragment string) string {
	return buildDoc("Саммари встречи", meta, contentFragment)
}

// TranscriptDoc builds the full Typst source for a transcript document.
func TranscriptDoc(meta interfaces.MeetingMeta, contentFragment string) string {
	return buildDoc("Транскрипция встречи", meta, contentFragment)
}

// Compile compiles a complete Typst document source to PDF bytes.
func Compile(ctx context.Context, fullDoc string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "typst-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	typFile := filepath.Join(dir, "doc.typ")
	if err := os.WriteFile(typFile, []byte(fullDoc), 0600); err != nil {
		return nil, fmt.Errorf("write typst: %w", err)
	}

	outFile := filepath.Join(dir, "doc.pdf")
	cmd := exec.CommandContext(ctx, "typst", "compile", typFile, outFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("typst compile: %w\n%s", err, out)
	}

	return os.ReadFile(outFile)
}

package tex

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Microservices/services/summarization-service/internal/application/interfaces"
)

// latexEscape escapes special LaTeX characters in plain text strings.
func latexEscape(s string) string {
	r := strings.NewReplacer(
		`\`, `\textbackslash{}`,
		`&`, `\&`,
		`%`, `\%`,
		`$`, `\$`,
		`#`, `\#`,
		`_`, `\_`,
		`{`, `\{`,
		`}`, `\}`,
		`~`, `\textasciitilde{}`,
		`^`, `\textasciicircum{}`,
	)
	return r.Replace(s)
}

const preamble = `\documentclass[12pt,a4paper]{article}

%% ── Encoding & language ───────────────────────────────────────────────────────
\usepackage[utf8]{inputenc}
\usepackage[T2A]{fontenc}
\usepackage[russian,english]{babel}

%% ── Fonts ─────────────────────────────────────────────────────────────────────
\usepackage{lmodern}
\usepackage{microtype}

%% ── Page geometry ─────────────────────────────────────────────────────────────
\usepackage[top=2.5cm, bottom=2.5cm, left=3cm, right=2.5cm]{geometry}

%% ── Colours ───────────────────────────────────────────────────────────────────
\usepackage[dvipsnames,table]{xcolor}
\definecolor{accent}{HTML}{2C5F8A}
\definecolor{muted}{HTML}{6B7280}
\definecolor{rulecol}{HTML}{D1D5DB}
\definecolor{headerbg}{HTML}{F3F6FA}

%% ── Header / footer ───────────────────────────────────────────────────────────
\usepackage{fancyhdr}
\pagestyle{fancy}
\fancyhf{}
\renewcommand{\headrulewidth}{0.4pt}
\renewcommand{\headrule}{\hbox to\headwidth{\color{rulecol}\leaders\hrule height \headrulewidth\hfill}}
\fancyhead[L]{\small\color{muted}\leftmark}
\fancyhead[R]{\small\color{muted}\thepage}

%% ── Spacing & typography ─────────────────────────────────────────────────────
\usepackage{setspace}
\setstretch{1.25}
\setlength{\parskip}{0.55em}
\setlength{\parindent}{0pt}

%% ── Section styling ──────────────────────────────────────────────────────────
\usepackage{titlesec}
\titleformat{\section}{\large\bfseries\color{accent}}{}{0em}{}[\vspace{-0.3em}\color{rulecol}\hrule\vspace{0.4em}]
\titleformat{\subsection}{\normalsize\bfseries\color{accent}}{}{0em}{}
\titlespacing{\section}{0pt}{1.4em}{0.6em}
\titlespacing{\subsection}{0pt}{1.0em}{0.3em}

%% ── Lists ────────────────────────────────────────────────────────────────────
\usepackage{enumitem}
\setlist[itemize]{topsep=0.3em, itemsep=0.2em, leftmargin=1.4em}

%% ── Misc ─────────────────────────────────────────────────────────────────────
\usepackage{booktabs}
\usepackage{array}
\usepackage{graphicx}
\usepackage{hyperref}
\hypersetup{colorlinks=true, linkcolor=accent, urlcolor=accent}

`

// metaBlock builds a styled metadata table from MeetingMeta.
func metaBlock(meta interfaces.MeetingMeta, title string) string {
	var sb strings.Builder

	// ── decorative top rule ──────────────────────────────────────────────────
	sb.WriteString(`{\color{accent}\rule{\linewidth}{1.5pt}}` + "\n\n")

	// ── document title ───────────────────────────────────────────────────────
	sb.WriteString(`{\Huge\bfseries\color{accent} ` + latexEscape(title) + "}\n\n")
	sb.WriteString(`{\color{accent}\rule{\linewidth}{0.6pt}}` + "\n\n")

	// ── metadata table ───────────────────────────────────────────────────────
	sb.WriteString(`\begin{tabular}{@{}>{\color{muted}\small}l @{\hspace{1em}} l}` + "\n")

	if meta.WorkspaceName != "" {
		sb.WriteString(`  \textbf{Воркспейс} & ` + latexEscape(meta.WorkspaceName) + ` \\` + "\n")
	}
	if meta.UploaderName != "" {
		sb.WriteString(`  \textbf{Загрузил} & ` + latexEscape(meta.UploaderName) + ` \\` + "\n")
	}
	if meta.Date != "" {
		sb.WriteString(`  \textbf{Дата} & ` + latexEscape(meta.Date) + ` \\` + "\n")
	}

	sb.WriteString(`\end{tabular}` + "\n\n")
	sb.WriteString(`{\color{rulecol}\rule{\linewidth}{0.4pt}}` + "\n\n")
	sb.WriteString(`\vspace{1em}` + "\n\n")

	return sb.String()
}

// SummaryDoc builds the full LaTeX source for a summary document.
func SummaryDoc(meta interfaces.MeetingMeta, contentFragment string) string {
	var sb strings.Builder
	sb.WriteString(preamble)
	sb.WriteString(`\begin{document}` + "\n\n")
	sb.WriteString(`\markboth{Саммари встречи}{}` + "\n\n")
	sb.WriteString(metaBlock(meta, "Саммари встречи"))
	sb.WriteString(contentFragment)
	sb.WriteString("\n\n" + `\end{document}` + "\n")
	return sb.String()
}

// TranscriptDoc builds the full LaTeX source for a transcript document.
func TranscriptDoc(meta interfaces.MeetingMeta, contentFragment string) string {
	var sb strings.Builder
	sb.WriteString(preamble)
	sb.WriteString(`\begin{document}` + "\n\n")
	sb.WriteString(`\markboth{Транскрипция встречи}{}` + "\n\n")
	sb.WriteString(metaBlock(meta, "Транскрипция встречи"))
	sb.WriteString(contentFragment)
	sb.WriteString("\n\n" + `\end{document}` + "\n")
	return sb.String()
}

// Compile compiles a complete LaTeX document source to PDF bytes.
func Compile(ctx context.Context, fullDoc string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "pdflatex-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	texFile := filepath.Join(dir, "doc.tex")
	if err := os.WriteFile(texFile, []byte(fullDoc), 0600); err != nil {
		return nil, fmt.Errorf("write tex: %w", err)
	}

	// Run twice so section marks / references resolve correctly.
	for range 2 {
		cmd := exec.CommandContext(ctx, "pdflatex",
			"-interaction=nonstopmode",
			"-output-directory", dir,
			texFile,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("pdflatex: %w\n%s", err, out)
		}
	}

	return os.ReadFile(filepath.Join(dir, "doc.pdf"))
}

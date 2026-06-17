package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Microservices/services/summarization-service/internal/application/interfaces"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type ClaudeSummarizer struct {
	client anthropic.Client
	model  anthropic.Model
}

func NewClaudeSummarizer(apiKey, baseURL, model string) *ClaudeSummarizer {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	return &ClaudeSummarizer{
		client: anthropic.NewClient(opts...),
		model:  anthropic.Model(model),
	}
}

type claudeResponse struct {
	Summary    string `json:"summary"`
	Transcript string `json:"transcript"`
}

func (c *ClaudeSummarizer) Summarize(ctx context.Context, segments []interfaces.Segment) (*interfaces.SummaryResult, error) {
	return c.SummarizeWithMeta(ctx, segments, interfaces.MeetingMeta{})
}

func (c *ClaudeSummarizer) SummarizeWithMeta(ctx context.Context, segments []interfaces.Segment, _ interfaces.MeetingMeta) (*interfaces.SummaryResult, error) {
	segmentsJSON, err := json.Marshal(segments)
	if err != nil {
		return nil, fmt.Errorf("marshal segments: %w", err)
	}

	prompt := fmt.Sprintf(`You are an assistant for processing meeting transcripts.
IMPORTANT: Detect the language of the transcript text and produce ALL output in that SAME language. Never switch to English if the transcript is in another language.

Given a transcript with segments (participantId, participantName, start, end, text), return ONLY a JSON object with two fields.
Do NOT include any document preamble, page setup, or metadata header — those are added separately.
Use Typst markup syntax.

- "summary": a Typst content fragment structured as:
  1. A "= <title>" heading in the transcript language (e.g. "= Ключевые тезисы") followed by a bullet list "- item" of key meeting takeaways.
  2. A "= <title>" heading (e.g. "= По участникам") with per-speaker "== <name>" subheadings (use the speaker's name exactly) followed by bullet lists of that speaker's key contributions.

- "transcript": a Typst content fragment with ALL segments in strict chronological order (sorted by start time).
  For EACH segment — one at a time, do not merge — emit exactly:
    #seg("SPEAKER_NAME", "M:SS -- M:SS", [segment text here])
  Where:
  - SPEAKER_NAME is participantName if available, otherwise participantId. Use the same name consistently.
  - M:SS is minutes:seconds formatted from the start/end float values (e.g. 0:05, 1:32, 10:07).
  - The segment text goes inside [...] as Typst content — escape # as \#, @ as \@.
  Do NOT group by speaker. Do NOT use headings. Just one #seg() call per segment in time order.

Typst escaping rules you MUST follow in ALL output:
  - Escape literal \# as \#, literal \@ as \@, literal \< as \<, literal \> as \>
  - Do NOT use any LaTeX syntax.

Return ONLY the raw JSON object, no markdown fences, no explanation.

Transcript:
%s`, string(segmentsJSON))

	adaptive := anthropic.ThinkingConfigAdaptiveParam{}
	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: 16000,
		Thinking:  anthropic.ThinkingConfigParamUnion{OfAdaptive: &adaptive},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude api: %w", err)
	}

	var text string
	for _, block := range resp.Content {
		if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
			text = tb.Text
			break
		}
	}

	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		if len(lines) >= 3 {
			text = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var cr claudeResponse
	if err := json.Unmarshal([]byte(text), &cr); err != nil {
		return nil, fmt.Errorf("parse claude response: %w (raw: %.500s)", err, text)
	}

	return &interfaces.SummaryResult{
		SummaryFragment:    cr.Summary,
		TranscriptFragment: cr.Transcript,
	}, nil
}

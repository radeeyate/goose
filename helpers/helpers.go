package helpers

import (
	"bytes"
	"fmt" // <-- Add fmt import if not already there
	"io"  // <-- Add io import
	"os"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/gohugoio/hugo-goldmark-extensions/extras"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

type MarkdownConfig struct {
	Theme                                 string
	SyntaxHighlightingUseCustomBackground bool
	SyntaxHighlightingCustomBackground    string
}

func IsFile(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	return !info.IsDir() && !info.Mode().IsDir(), nil
}

func IsDir(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

func RemoveDuplicates(input []interface{}) []interface{} {
	seen := make(map[interface{}]bool)
	result := []interface{}{}
	for _, item := range input {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

func RenderMarkdown(input string, config MarkdownConfig) (string, error) {
	var buf bytes.Buffer
	mdRenderer := generateMarkdownRenderer(config)
	err := mdRenderer.Convert([]byte(input), &buf)
	if err != nil {
		return "", fmt.Errorf("markdown conversion failed: %w", err)
	}
	return buf.String(), nil
}

func ExtractMetadata(
	input string,
	config MarkdownConfig,
	defaultMeta map[string]interface{},
) map[string]interface{} {
	context := parser.NewContext()
	mdRenderer := generateMarkdownRenderer(config)
	_ = mdRenderer.Convert([]byte(input), io.Discard, parser.WithContext(context))
	pageMeta := meta.Get(context)

	finalMeta := make(map[string]interface{})
	if defaultMeta != nil {
		for key, value := range defaultMeta {
			finalMeta[key] = value
		}
	}

	if pageMeta != nil {
		for key, value := range pageMeta {
			finalMeta[key] = value
		}
	}

	return finalMeta
}

func generateMarkdownRenderer(config MarkdownConfig) goldmark.Markdown {
	var highlightingConfig map[chroma.TokenType]string
	if config.SyntaxHighlightingUseCustomBackground {
		highlightingConfig = map[chroma.TokenType]string{
			chroma.Background: config.SyntaxHighlightingCustomBackground,
		}
	}

	return goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Linkify,
			extension.Footnote,
			extension.Typographer,
			extension.Strikethrough,
			extension.TaskList,
			extension.DefinitionList,
			highlighting.NewHighlighting(
				highlighting.WithStyle(config.Theme),
				highlighting.WithFormatOptions(
					chromahtml.WithLineNumbers(true),
					chromahtml.WithCustomCSS(highlightingConfig),
				),
				highlighting.WithGuessLanguage(true),
			),
			emoji.New(emoji.WithRenderingMethod(emoji.Twemoji)),
			meta.Meta,
			extras.New(
				extras.Config{
					Delete:      extras.DeleteConfig{Enable: true},
					Insert:      extras.InsertConfig{Enable: true},
					Mark:        extras.MarkConfig{Enable: true},
					Subscript:   extras.SubscriptConfig{Enable: true},
					Superscript: extras.SuperscriptConfig{Enable: true},
				},
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)
}

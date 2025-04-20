package helpers

import (
	"bytes"
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

func GenerateMarkdown(input, theme string) (string, map[string]interface{}) {
	var buf bytes.Buffer

	err := generateMarkdownRenderer(theme).Convert([]byte(input), &buf)
	if err != nil {
		panic(err)
	}

	metadata := ExtractMetadata(input, theme)
	return buf.String(), metadata
}

func ExtractMetadata(input string, theme string) map[string]interface{} {
	context := parser.NewContext()
	generateMarkdownRenderer(
		theme,
	).Convert([]byte(input), &bytes.Buffer{}, parser.WithContext(context))
	metadata := meta.Get(context)
	return metadata
}

func generateMarkdownRenderer(theme string) goldmark.Markdown {
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
				highlighting.WithStyle(theme),
				highlighting.WithFormatOptions(
					chromahtml.WithLineNumbers(true),
					chromahtml.WithCustomCSS(map[chroma.TokenType]string{
						chroma.Background: "background-color: #3e4451;",
					}),
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

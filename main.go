package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"golang.org/x/net/html"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/gohugoio/hugo-goldmark-extensions/extras"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"

	"github.com/tdewolff/minify/v2"
	minifyhtml "github.com/tdewolff/minify/v2/html"
	minifycss "github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"
)

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.Linkify,
		extension.Footnote,
		extension.Typographer,
		extension.Strikethrough,
		extension.TaskList,
		extension.DefinitionList,
		highlighting.NewHighlighting(
			highlighting.WithStyle("monokai"),
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

func makeMD(input string) (string, map[string]interface{}) {
	var buf bytes.Buffer

	context := parser.NewContext()
	err := md.Convert([]byte(input), &buf, parser.WithContext(context))
	if err != nil {
		panic(err)
	}

	metadata := meta.Get(context)

	return buf.String(), metadata
}

func replaceMetaPlaceholders(markdown string, metadata map[string]interface{}) string {
	re := regexp.MustCompile(`{{\s*\.meta\.([a-zA-Z0-9_-]+)\s*}}`)

	return re.ReplaceAllStringFunc(markdown, func(match string) string {
		key := re.FindStringSubmatch(match)[1]
		if value, ok := metadata[key]; ok {
			return fmt.Sprintf("%v", value)
		}
		return match
	})
}

func main() {
	srcFolder := "."

	srcInfo, err := os.Stat(srcFolder)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Source folder not found.")
			return
		}
		log.Fatal(err)
	}
	if !srcInfo.IsDir() {
		fmt.Println("Source folder is not a directory.")
		return
	}

	pagesDir := filepath.Join(srcFolder, "source", "pages")
	_, err = os.Stat(pagesDir) // if source/pages exists
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("source/pages directory not found.")
			return
		}
		log.Fatal(err)
	}

	buildDir := filepath.Join(srcFolder, "build")

	err = os.RemoveAll(buildDir)
	if err != nil && !os.IsNotExist(err) {
		log.Println("Warning: Could not remove existing build directory:", err)
	}

	err = os.MkdirAll(buildDir, 0755)
	if err != nil && !os.IsExist(err) {
		log.Fatal("Error creating build directory:", err)
	}

	err = filepath.Walk(pagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing path %q: %v\n", path, err)
			return err
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".md" {
			return nil
		}

		fmt.Printf("File found: %s... ", path)

		code, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Error reading file %s: %v\n", path, err)
			return nil
		}

		relPath, err := filepath.Rel(pagesDir, path)
		if err != nil {
			log.Printf("Error calculating relative path for %s: %v\n", path, err)
			return nil
		}

		baseName := filepath.Base(path)
		ext := filepath.Ext(baseName)
		nameWithoutExt := baseName[:len(baseName)-len(ext)]
		outPath := filepath.Join(buildDir, filepath.Dir(relPath), nameWithoutExt+".html")

		err = os.MkdirAll(filepath.Dir(outPath), 0755)
		if err != nil {
			log.Printf("Error creating directory %s: %v\n", filepath.Dir(outPath), err)
			return nil
		}

		out, err := os.Create(outPath)
		if err != nil {
			log.Printf("Error creating output file %s: %v\n", outPath, err)
			return nil
		}
		defer out.Close()

		markdown, metadata := makeMD(string(code))
		markdown = replaceMetaPlaceholders(markdown, metadata)

		template, err := os.ReadFile("source/templates/default.html")
		if err != nil {
			panic(err)
		}

		css, err := os.ReadFile("source/styles/default.css")
		if err != nil {
			panic(err)
		}

		doc, err := html.Parse(bytes.NewReader(template))
		if err != nil {
			panic(err)
		}

		title := "wah"

		var walk func(*html.Node)
		walk = func(n *html.Node) {
			if n.Type == html.ElementNode {
				if n.Data == "head" {
					titleNode := &html.Node{
						Type: html.ElementNode,
						Data: "title",
						FirstChild: &html.Node{
							Type: html.TextNode,
							Data: title,
						},
					}
					n.AppendChild(titleNode)

					styleNode := &html.Node{
						Type: html.ElementNode,
						Data: "style",
						FirstChild: &html.Node{
							Type: html.TextNode,
							Data: string(css),
						},
					}
					n.AppendChild(styleNode)
				}
			}

			for child := n.FirstChild; child != nil; child = child.NextSibling {
				walk(child)
			}
		}

		walk(doc)

		var buf bytes.Buffer
		err = html.Render(&buf, doc)
		if err != nil {
			panic(err)
		}

		renderedHtml := buf.String()
		renderedHtml = strings.ReplaceAll(renderedHtml, "markdown", markdown)

		minifier := minify.New()
		htmlMinifier := &minifyhtml.Minifier{
			KeepDocumentTags:        true,
			KeepEndTags:             false,
			KeepConditionalComments: false,
			KeepQuotes:              false,
			KeepWhitespace:          false,
		}
		minifier.AddFunc("text/html", htmlMinifier.Minify)
		minifier.AddFunc("text/css", minifycss.Minify)
		minifier.AddFunc("text/javascript", js.Minify)

		minifiedHtml, err := minifier.String("text/html", renderedHtml)
		if err != nil {
			panic(err)
		}

		_, err = out.WriteString(minifiedHtml)
		if err != nil {
			log.Printf("Error writing to output file %s: %v\n", outPath, err)
		}

		fmt.Println("generated.")
		return nil
	})

	if err != nil {
		log.Printf("Error walking the path %q: %v\n", pagesDir, err)
	}
}

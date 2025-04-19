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
	minifycss "github.com/tdewolff/minify/v2/css"
	minifyhtml "github.com/tdewolff/minify/v2/html"
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

	err := md.Convert([]byte(input), &buf)
	if err != nil {
		panic(err)
	}

	metadata := extractMetadata(input)
	return buf.String(), metadata
}

func extractMetadata(input string) map[string]interface{} {
	context := parser.NewContext()
	md.Convert([]byte(input), &bytes.Buffer{}, parser.WithContext(context))
	metadata := meta.Get(context)
	return metadata
}

func replaceMetaPlaceholders(
	markdown string,
	metadata map[string]interface{},
	fileRootDir, rootDir string,
) string {
	re := regexp.MustCompile(`{{\s*\.meta\.([a-zA-Z0-9_-]+)\s*}}`)
	reFromFile := regexp.MustCompile(`{{\s*from\s+([^\s]+)\s+\.meta\.([a-zA-Z0-9_-]+)\s*}}`)

	markdown = re.ReplaceAllStringFunc(markdown, func(match string) string {
		key := re.FindStringSubmatch(match)[1]
		if value, ok := metadata[key]; ok {
			return fmt.Sprintf("%v", value)
		}
		return match
	})

	markdown = reFromFile.ReplaceAllStringFunc(markdown, func(match string) string {
		matches := reFromFile.FindStringSubmatch(match)
		if len(matches) != 3 {
			return match // invalid format
		}

		filePath := matches[1]

		if !strings.HasSuffix(filePath, ".md") {
			filePath += ".md"
		}

		metaKey := matches[2]

		// resolve relative path based on the current file's directory
		absFilePath := filepath.Join(fileRootDir, filePath)

		// detect if absfilepath is outside root directory of pages (of the whole project, not the root of the file)
		if !strings.HasPrefix(absFilePath, rootDir) {
			log.Printf("Error: File %s is outside the root directory %s\n", absFilePath, rootDir)
			return match
		}

		content, err := os.ReadFile(absFilePath)
		if err != nil {
			log.Printf("Error reading file %s: %v\n", absFilePath, err)
			return match
		}

		_, referencedMetadata := makeMD(string(content))

		if value, ok := referencedMetadata[metaKey]; ok {
			return fmt.Sprintf("%v", value)
		}
		return match
	})

	return markdown
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

		if exists, err := exists(filepath.Join(pagesDir, filepath.Dir(relPath), nameWithoutExt, "index.md")); err != nil ||
			exists {
			log.Printf(
				"Both \"%s\" and \"%s\" exist; skipping.\n",
				baseName,
				filepath.Join(filepath.Dir(relPath), nameWithoutExt, "index.md"),
			)
			return nil
		} else {
			if nameWithoutExt != "index" {
				outPath = filepath.Join(buildDir, filepath.Dir(relPath), nameWithoutExt, "index.html")
			}
		}

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
		markdown = replaceMetaPlaceholders(markdown, metadata, filepath.Dir(path), pagesDir)

		var title string
		if metadata["title"] != nil {
			title = fmt.Sprintf("%v", metadata["title"])
		}

		var css []byte
		if metadata["styles"] != nil {
			for _, style := range removeDuplicates(metadata["styles"].([]interface{})) {
				fileName := filepath.Base(style.(string))
				if !strings.HasSuffix(fileName, ".css") {
					fileName += ".css"
				}

				if existence, err := exists(filepath.Join(srcFolder, "source", "styles", fileName)); existence ||
					err != nil {
					addedCss, err := os.ReadFile(
						filepath.Join(srcFolder, "source", "styles", fileName),
					)

					if err != nil {
						panic(err)
					}

					css = append(css, append(addedCss, "\n"...)...)
				} else {
					log.Printf("Stylesheet %s does not exist.", fileName)
				}
			}
		} else {
			if existence, err := exists("source/styles/default.css"); existence || err != nil {
				css, err = os.ReadFile(
					"source/styles/default.css",
				)
				if err != nil {
					panic(err)
				}
			} else {
				log.Printf("Default stylesheet does not exist; proceeding to not use a stylesheet.")
			}
		}

		var scripts [][]byte
		if metadata["scripts"] != nil {
			for _, style := range removeDuplicates(metadata["scripts"].([]interface{})) {
				fileName := filepath.Base(style.(string))
				if !strings.HasSuffix(fileName, ".js") {
					fileName += ".js"
				}

				if existence, err := exists(filepath.Join(srcFolder, "source", "scripts", fileName)); existence ||
					err != nil {
					addedJS, err := os.ReadFile(
						filepath.Join(srcFolder, "source", "scripts", fileName),
					)

					if err != nil {
						panic(err)
					}

					fmt.Println(string(addedJS))

					scripts = append(scripts, addedJS)
				} else {
					log.Printf("Script %s does not exist.", fileName)
				}
			}
		} else {
			if existence, err := exists("source/scripts/default.js"); existence || err != nil {
				defaultScript, err := os.ReadFile(
					"source/scripts/default.js",
				)
				if err != nil {
					panic(err)
				}
				scripts = append(scripts, defaultScript)
			} else {
				log.Printf("Default script does not exist; proceeding to not use a script.")
			}
		}

		var template []byte
		if existence, err := exists("source/templates/default.html"); existence || err != nil {
			if metadata["template"] != nil {
				fileName := filepath.Base(metadata["template"].(string))
				if !strings.HasSuffix(fileName, ".html") {
					fileName += ".html"
				}

				if existence, err := exists(filepath.Join(srcFolder, "source", "templates", fileName)); existence ||
					err != nil {
					template, err = os.ReadFile(
						filepath.Join(srcFolder, "source", "templates", fileName),
					)
					if err != nil {
						panic(err)
					}
				} else {
					log.Printf("Template %s does not exist.", fileName)
				}
			} else {
				template, err = os.ReadFile(
					"source/templates/default.html",
				)
				if err != nil {
					panic(err)
				}
			}
		} else {
			template = []byte("markdown")
			log.Printf("Default template does not exist; proceeding to not use a template.")
		}

		doc, err := html.Parse(bytes.NewReader(template))
		if err != nil {
			panic(err)
		}

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

					for _, script := range scripts {
						scriptsNode := &html.Node{
							Type: html.ElementNode,
							Data: "script",
							FirstChild: &html.Node{
								Type: html.TextNode,
								Data: string(script),
							},
						}
						n.AppendChild(scriptsNode)
					}

					scriptNode := &html.Node{
						Type: html.ElementNode,
						Data: "script",
						Attr: []html.Attribute{
							{
								Key: "src",
								Val: "https://unpkg.com/htmx.org@2.0.4",
							},
						},
					}
					n.AppendChild(scriptNode)
				} else if n.Data == "a" {
					n.Attr = append(n.Attr, html.Attribute{
						Key: "hx-boost",
						Val: "true",
					})
				} else if n.Data == "markdown" {
					// replace with the markdown
					
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

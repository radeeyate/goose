package cmd

import (
	"html/template"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/net/html"

	"github.com/tdewolff/minify/v2"
	minifycss "github.com/tdewolff/minify/v2/css"
	minifyhtml "github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"

	"github.com/radeeyate/goose/helpers"
)

func init() {
	generateCmd.Run = runGenerate
}

func runGenerate(cmd *cobra.Command, args []string) {
	fmt.Println("Starting static site generation...")

	sourceDir := viper.GetString("sourceDir")
	buildDir := viper.GetString("buildDir")
	stylesDir := filepath.Join(sourceDir, viper.GetString("stylesDir"))
	scriptsDir := filepath.Join(sourceDir, viper.GetString("scriptsDir"))
	pagesDir := filepath.Join(sourceDir, viper.GetString("pagesDir"))
	templatesDir := filepath.Join(sourceDir, viper.GetString("templatesDir"))
	staticDir := filepath.Join(sourceDir, viper.GetString("staticDir"))
	syntaxHighlightingStyle := viper.GetString("syntaxHighlightingStyle")
	defaultTemplate := viper.GetString("defaultTemplate")
	defaultStyles := viper.GetStringSlice("defaultStyles")
	defaultScripts := viper.GetStringSlice("defaultScripts")
	minifyOutput := viper.GetBool("minifyOutput")
	enableHtmx := viper.GetBool("enableHtmx")
	addHxBoost := viper.GetBool("addHxBoost")
	htmxSourceURL := viper.GetString("htmxSourceURL")
	includeDrafts := viper.GetBool("includeDrafts")
	//markdownPlaceholderTag := viper.GetString("markdownPlaceholderTag")
	prettyURLs := viper.GetBool("prettyURLs")
	defaultMetadata := viper.Get("defaultMetadata").(map[string]interface{})
	syntaxHighlightingUseCustomBackground := viper.GetBool("syntaxHighlightingUseCustomBackground")
	syntaxHighlightingCustomBackground := viper.GetString("syntaxHighlightingCustomBackground")
	enableCodeBlockLineNumbers := viper.GetBool("enableCodeBlockLineNumbers")
	enableEmoji := viper.GetBool("enableEmoji")

	if syntaxHighlightingUseCustomBackground && syntaxHighlightingCustomBackground == "" {
		log.Println(
			"Warning: syntaxHighlightingUseCustomBackground is set to true, but no custom background color was provided. Using default.",
		)
	}

	if exists, err := helpers.IsDir(sourceDir); !exists && err != nil {
		fmt.Printf("%s directory not found.", sourceDir)
		return
	}

	if exists, err := helpers.IsDir(pagesDir); !exists && err != nil {
		log.Fatalf("%s/%s directory not found.", sourceDir, pagesDir)
	}

	err := os.RemoveAll(buildDir)
	if err != nil && !os.IsNotExist(err) {
		log.Println("Warning: Could not remove existing build directory:", err)
	}

	err = os.MkdirAll(buildDir, 0755)
	if err != nil && !os.IsExist(err) {
		log.Fatal("Error creating build directory:", err)
	}

	if exists, err := helpers.IsDir(staticDir); exists && err == nil {
		err = os.CopyFS(filepath.Join(buildDir, "static"), os.DirFS(staticDir))
		if err != nil {
			panic(err)
		}
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

		metadata := helpers.ExtractMetadata(
			string(code),
			helpers.MarkdownConfig{
				Theme:                                 syntaxHighlightingStyle,
				SyntaxHighlightingUseCustomBackground: false,
				SyntaxHighlightingCustomBackground:    "",
				EnableCodeBlockLineNumbers:            enableCodeBlockLineNumbers,
				EnableEmoji:                           enableEmoji,
			},
			defaultMetadata,
		)
		if !includeDrafts && metadata["draft"] == true { // skip draft
			log.Println("Skipping draft.")
			return nil
		}

		baseName := filepath.Base(path)
		ext := filepath.Ext(baseName)
		nameWithoutExt := baseName[:len(baseName)-len(ext)]
		outPath := filepath.Join(buildDir, filepath.Dir(relPath), nameWithoutExt+".html")

		if prettyURLs {
			if exists, err := helpers.IsFile(filepath.Join(pagesDir, filepath.Dir(relPath), nameWithoutExt, "index.md")); exists &&
				err == nil ||
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

		markdown, err := helpers.RenderMarkdown(
			string(code),
			helpers.MarkdownConfig{
				Theme:                                 syntaxHighlightingStyle,
				SyntaxHighlightingUseCustomBackground: syntaxHighlightingUseCustomBackground,
				SyntaxHighlightingCustomBackground:    syntaxHighlightingCustomBackground,
				EnableCodeBlockLineNumbers:            enableCodeBlockLineNumbers,
				EnableEmoji:                           enableEmoji,
			},
		)
		if err != nil {
			log.Printf("Error rendering markdown for file %s: %v\n", path, err)
			return nil
		}

		markdown = replaceMetaPlaceholders(
			markdown,
			defaultMetadata,
			metadata,
			helpers.MarkdownConfig{
				Theme:                                 syntaxHighlightingStyle,
				SyntaxHighlightingUseCustomBackground: syntaxHighlightingUseCustomBackground,
				SyntaxHighlightingCustomBackground:    syntaxHighlightingCustomBackground,
				EnableCodeBlockLineNumbers:            enableCodeBlockLineNumbers,
				EnableEmoji:                           enableEmoji,
			},
			filepath.Dir(path),
			pagesDir,
		)

		var title string
		if metadata["title"] != nil {
			title = fmt.Sprintf("%v", metadata["title"])
		}

		var css []byte
		if metadata["styles"] != nil {
			for _, style := range helpers.RemoveDuplicates(metadata["styles"].([]interface{})) {
				fileName := filepath.Base(style.(string))
				if !strings.HasSuffix(fileName, ".css") {
					fileName += ".css"
				}

				if exists, err := helpers.IsFile(filepath.Join(stylesDir, fileName)); exists &&
					err == nil {
					addedCSS, err := os.ReadFile(
						filepath.Join(stylesDir, fileName),
					)

					if err != nil {
						panic(err)
					}

					css = append(css, append(addedCSS, "\n"...)...)
				} else {
					log.Printf("Stylesheet %s does not exist.", fileName)
				}
			}
		} else {
			for _, style := range defaultStyles {
				if exists, err := helpers.IsFile(filepath.Join(stylesDir, style)); exists && err == nil {
					addedCSS, err := os.ReadFile(
						filepath.Join(stylesDir, style),
					)
					if err != nil {
						panic(err)
					}

					css = append(css, append(addedCSS, "\n"...)...)
				} else {
					log.Printf("Default stylesheet %s does not exist; proceeding to not use a stylesheet.", style)
				}
			}
		}

		var scripts [][]byte
		if metadata["scripts"] != nil {
			for _, style := range helpers.RemoveDuplicates(metadata["scripts"].([]interface{})) {
				fileName := filepath.Base(style.(string))
				if !strings.HasSuffix(fileName, ".js") {
					fileName += ".js"
				}

				if exists, err := helpers.IsFile(filepath.Join(scriptsDir, fileName)); exists &&
					err == nil {
					addedJS, err := os.ReadFile(
						filepath.Join(scriptsDir, fileName),
					)

					if err != nil {
						panic(err)
					}

					scripts = append(scripts, addedJS)
				} else {
					log.Printf("Script %s does not exist.", fileName)
				}
			}
		} else {
			for _, script := range defaultScripts {
				if exists, err := helpers.IsFile(filepath.Join(scriptsDir, script)); exists && err == nil {
					addedJS, err := os.ReadFile(
						filepath.Join(scriptsDir, script),
					)
					if err != nil {
						panic(err)
					}

					scripts = append(scripts, addedJS)
				} else {
					log.Printf("Default script %s does not exist; proceeding to not use a script.", script)
				}
			}
		}

		var templateBytes []byte
		if exists, err := helpers.IsFile(filepath.Join(templatesDir, defaultTemplate)); exists &&
			err == nil {
			if metadata["template"] != nil {
				fileName := filepath.Base(metadata["template"].(string))
				if !strings.HasSuffix(fileName, ".html") {
					fileName += ".html"
				}

				if exists, err := helpers.IsFile(filepath.Join(templatesDir, fileName)); exists &&
					err == nil {
					templateBytes, err = os.ReadFile(
						filepath.Join(templatesDir, fileName),
					)
					if err != nil {
						panic(err)
					}
				} else {
					log.Printf("Template %s does not exist.", fileName)
				}
			} else {
				templateBytes, err = os.ReadFile(
					filepath.Join(templatesDir, defaultTemplate),
				)
				if err != nil {
					panic(err)
				}
			}
		} else {
			templateBytes = []byte(`<!doctypehtml><html lang="en"><meta charset="UTF-8"><meta content="width=device-width,initial-scale=1"name="viewport"><body><markdown></markdown>`)
			log.Printf("Default template does not exist; proceeding to not use a template.")
		}

		doc, err := html.Parse(bytes.NewReader(templateBytes))
		if err != nil {
			panic(err)
		}

		var walk func(*html.Node)
		walk = func(n *html.Node) {
			if n.Type == html.ElementNode {
				switch n.Data {
				case "head":
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

					if enableHtmx {
						scriptNode := &html.Node{
							Type: html.ElementNode,
							Data: "script",
							Attr: []html.Attribute{
								{
									Key: "src",
									Val: htmxSourceURL,
								},
							},
						}
						n.AppendChild(scriptNode)
					}
				case "a":
					if addHxBoost {
						n.Attr = append(n.Attr, html.Attribute{
							Key: "hx-boost",
							Val: "true",
						})
					}
				/*case markdownPlaceholderTag:
					if n.Parent == nil {
						log.Printf("Warning: <markdown> tag found without parent in %s", path)
						break
					}

					markdownNodes, parseErr := html.ParseFragment(
						bytes.NewReader([]byte(markdown)),
						n.Parent,
					)
					if parseErr != nil {
						log.Printf(
							"Error parsing markdown HTML fragment for %s: %v. Inserting raw content.",
							path,
							parseErr,
						)

						errorNode := &html.Node{
							Type: html.ElementNode,
							Data: "div",
							Attr: []html.Attribute{
								{
									Key: "style",
									Val: "color:red; border: 1px solid red; padding: 1em;",
								},
							},
							FirstChild: &html.Node{
								Type: html.TextNode,
								Data: fmt.Sprintf(
									"Error processing markdown content: %v",
									parseErr,
								),
							},
						}
						n.Parent.InsertBefore(errorNode, n)
					} else {
						for _, newNode := range markdownNodes {
							n.Parent.InsertBefore(newNode, n)
						}
					}

					n.Parent.RemoveChild(n)*/
				}
			}

			for c := n.FirstChild; c != nil; {
				next := c.NextSibling
				walk(c)
				c = next
			}
		}

		walk(doc)

		var buf bytes.Buffer
		err = html.Render(&buf, doc)
		if err != nil {
			panic(err)
		}

		renderedHtml := buf.String()

		tmpl, err := template.New("").Parse(renderedHtml)
		if err != nil {
			// do something
			panic(err)
		}
		var output bytes.Buffer
		data := map[any]any{"Markdown": template.HTML(markdown)}
		for k, v := range metadata {
			data[k] = v
		}
		if err := tmpl.Execute(&output, data); err != nil {
			panic(err)
		}


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

		var minifedHTML string
		if minifyOutput {
			minifedHTML, err = minifier.String("text/html", output.String())
			if err != nil {
				panic(err)
			}
		} else {
			minifedHTML = output.String()
		}

		_, err = out.WriteString(minifedHTML)
		if err != nil {
			log.Printf("Error writing to output file %s: %v\n", outPath, err)
		}

		fmt.Println("generated.")
		return nil
	})

	if err != nil {
		log.Printf("Error walking the path %q: %v\n", pagesDir, err)
	}

	fmt.Println("\nStatic site generation complete!")
}

func replaceMetaPlaceholders(
	markdown string,
	defaultMetadata, metadata map[string]interface{}, config helpers.MarkdownConfig,
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

		referencedMetadata := helpers.ExtractMetadata(
			string(content),
			config,
			defaultMetadata,
		)

		if value, ok := referencedMetadata[metaKey]; ok {
			return fmt.Sprintf("%v", value)
		}
		return match
	})

	return markdown
}

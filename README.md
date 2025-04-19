# goose

goose is an opinionated static site generator written in Golang.

## Installation

To get started, start by cloning the repository:

```sh
git clone https://github.com/radeeyate/goose.git
cd goose
```

## Usage

The directory structure of a goose project is as follows:

```text
source
├── pages
│   ├── blog
│   │   └── firstblog.md
│   ├── blog.md
│   ├── index.md
│   ├── contact
│   │   └── index.md
│   └── test.md
├── styles
│   └── default.css
└── templates
    └── default.html
```

- All files are read from the `source` directory. This will be configurable in the future.
- The default template is stored in the `templates` directory as `default.html`. In the future, this will be configured via a value in the Markdown file's front matter.
- The default style is stored in the `styles` directory as `default.css`.
- All pages are read from the `pages` directory.
  - `index.md` -> `index.html`
  - `blog.md` -> `blog/index.html` (unless `blog/index.md` already exists)
  - `blog/firstblog.md` -> `blog/firstblog.html`

To generate the static site, run the following command:

```sh
go run main.go
```

This will generate a static site in the `build` directory.

## Features

- [x] Markdown support
- [x] Templating
- [x] Metadata variable placeholders
- [x] CSS bundling
- [x] JS bundling

### Markdown support

- Relative links
- Absolute links
- Images
- Code blocks
- Headers
- Lists
- Bold
- Italic
- Strikethrough
- Blockquotes
- Horizontal rules
- Github Flavored Markdown
- Plain text -> URL Conversion
- Footnotes
- Definitions
- Task Lists
- Code Highlighting
- Emojis :rocket:
- Superscript & Subscript
- Text Highlighting

### Templating

Anywhere the word `markdown` is found in the template, your Markdown will be inserted. In the future, you will be able to have a `<markdown />` tag instead.

The template for a page is determined by the `template` variable in the front matter:

```md
---
title: My Page
template: custom
---
```

In this example, the `custom.html` file in the `templates` directory will be used as the template. If no template is declared, `default.html` will be used.

The output of all Markdown conversion and Markdown conversion is minified, including CSS and Javascript.

### CSS Bundling

The stylesheet for a declared page will be automatically inserted into the `<head>` of the HTML template. You can declare custom styles for a page using the front matter:

```md
---
title: My Page
styles:
- default
- custom
---
```

In this example, the `default.css` and `custom.css` files will be bundled into the page. You can place these styles in the `styles` directory as mentioned earlier. `default.css` will be applied if a custom style is not declared.

### JS Bundling

The script for a declared page will be automatically inserted into the `<head>` of the HTML template. You can declare custom scripts for a page using the front matter:

```md
---
title: My Page
scripts:
- default
- custom
---
```

In this example, the `default.js` and `custom.js` files will be bundled into the page. You can place these styles in the `scripts` directory as mentioned earlier. `default.js` will be applied if a custom style is not declared.

### Metadata

You can access variables in the front matter of a Markdown file using the `{{ .meta.<variable_name> }}` syntax. You can also access variables from other files using the `{{ from <path> .meta.<variable_name> }}` syntax. 

For example, you could use `{{ .meta.title }}` to retrieve the document's title. You can use `{{ from index .meta.title }}` to retrieve the title of a file named `index.md`. This supports directory traversal, so you could do something like `{{ from ../index .meta.title }}` or `{{ from blogs/firstblog .meta.tags }}`

If a `title` variable is found in the front matter of a Markdown file, it is automatically inserted into the document's `<head>`.
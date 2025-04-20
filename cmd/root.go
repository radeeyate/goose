package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

const (
	defaultSourceDir     = "source"
	defaultBuildDir      = "build"
	defaultPagesDir      = "pages"
	defaultStylesDir     = "styles"
	defaultScriptsDir    = "scripts"
	defaultTemplatesDir  = "templates"
	defaultStaticDir     = "static"
	defaultTemplate      = "default.html"
	defaultStyle         = "github"
	defaultHtmxSourceURL = "https://unpkg.com/htmx.org@2.0.4"
)

func init() {

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "config file (default is ./goose.toml or $HOME/.goose/goose.toml)")

	rootCmd.PersistentFlags().
		StringP("source", "s", defaultSourceDir, "Source directory containing website content")
	rootCmd.PersistentFlags().
		StringP("build", "b", defaultBuildDir, "Directory to output the generated website")

	viper.BindPFlag("sourceDir", rootCmd.PersistentFlags().Lookup("source"))
	viper.BindPFlag("buildDir", rootCmd.PersistentFlags().Lookup("build"))

	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(serveCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(".")
		viper.AddConfigPath(filepath.Join(home, ".goose"))
		viper.SetConfigName("goose")
		viper.SetConfigType("toml")
	}

	viper.SetDefault("sourceDir", defaultSourceDir)
	viper.SetDefault("buildDir", defaultBuildDir)
	viper.SetDefault("pagesDir", defaultPagesDir)
	viper.SetDefault("stylesDir", defaultStylesDir)
	viper.SetDefault("scriptsDir", defaultScriptsDir)
	viper.SetDefault("templatesDir", defaultTemplatesDir)
	viper.SetDefault("staticDir", defaultStaticDir)
	viper.SetDefault("syntaxHighlightingStyle", defaultStyle)
	viper.SetDefault("defaultTemplate", defaultTemplate)
	viper.SetDefault("defaultStyles", []string{"default.css"})
	viper.SetDefault("defaultScripts", []string{"default.js"})
	viper.SetDefault("minifyOutput", true)
	viper.SetDefault("enableHtmx", true)
	viper.SetDefault("addHxBoost", true)
	viper.SetDefault("htmxSourceURL", defaultHtmxSourceURL)
	viper.SetDefault("includeDrafts", false)
	viper.SetDefault("markdownPlaceholderTag", "markdown")
	viper.SetDefault("prettyURLs", true)
	viper.SetDefault("defaultMetadata", map[string]interface{}{})
	viper.SetDefault("syntaxHighlightingUseCustomBackground", false)
	viper.SetDefault("syntaxHighlightingCustomBackground", "")
	viper.SetDefault("enableCodeBlockLineNumbers", true)
	viper.SetDefault("enableEmoji", true)

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("Config file not found, using defaults/flags.")
		} else {
			fmt.Printf("Error reading config file %s: %v\n", viper.ConfigFileUsed(), err)
		}
	}
}

var rootCmd = &cobra.Command{
	Use:   "goose",
	Short: "goose is a static site generator",
	Long:  `goose is a static site generator written in Go.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate the static site",
	Long:  `Generate the static site from the source files.`,
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts a development server",
	Long:  `Starts a local development server that serves the built site and watches for changes in the source directory to rebuild automatically.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
)

const regexFolder = "Reg"
const version = "1.0.0"

type GrepResult struct {
	Source  string
	Pattern string
	File    string
	Line    string
	Match   string
}

func main() {
	printBanner()

	if len(os.Args) < 4 {
		printUsage()
		os.Exit(1)
	}

	var (
		lang           string
		searchPath     string
		excludedExts   []string
		includeSecrets bool
	)

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "-L", "-l":
			if i+1 >= len(os.Args) {
				color.Red("❌ Error: Missing value after -L")
				printUsage()
				os.Exit(1)
			}
			lang = os.Args[i+1]
			i++
		case "--secrets":
			includeSecrets = true
		case "--exclude":
			if i+1 >= len(os.Args) {
				color.Red("❌ Error: Missing value after --exclude")
				os.Exit(1)
			}
			parts := strings.Split(os.Args[i+1], ",")
			for _, ext := range parts {
				excludedExts = append(excludedExts, strings.TrimSpace(ext))
			}
			i++
		default:
			if strings.HasPrefix(arg, "-") {
				color.Red("❌ Unknown flag: %s", arg)
				printUsage()
				os.Exit(1)
			} else if searchPath == "" {
				searchPath = arg
			} else {
				color.Red("❌ Unexpected argument: %s", arg)
				printUsage()
				os.Exit(1)
			}
		}
	}

	if lang == "" || searchPath == "" {
		color.Red("❌ Error: Missing required arguments")
		printUsage()
		os.Exit(1)
	}

	var langs []string
	var err error

	if lang == "all" {
		langs, err = getAllLanguages()
		if err != nil {
			color.Red("❌ Failed to list languages: %v", err)
			os.Exit(1)
		}
	} else {
		langs = []string{lang}
	}

	if includeSecrets {
		langs = append(langs, "secrets")
	}

	var results []GrepResult

	for _, l := range langs {
		patterns, err := loadPatterns(l)
		if err != nil {
			color.Yellow("⚠️  Skipping %s: %v", l, err)
			continue
		}
		for _, pattern := range patterns {
			found, err := collectGrepResults(l, pattern, searchPath, excludedExts)
			if err != nil {
				color.Red("❌ Error running grep for pattern %s: %v", pattern, err)
				continue
			}
			results = append(results, found...)
		}
	}

	printResultsPretty(results)
}

func loadPatterns(language string) ([]string, error) {
	fileName := filepath.Join(regexFolder, fmt.Sprintf("%s.json", language))
	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", fileName, err)
	}
	var patterns []string
	if err := json.Unmarshal(data, &patterns); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", fileName, err)
	}
	return patterns, nil
}

func getAllLanguages() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(regexFolder, "*.json"))
	if err != nil {
		return nil, err
	}
	var langs []string
	for _, f := range files {
		base := filepath.Base(f)
		lang := strings.TrimSuffix(base, ".json")
		if lang != "secrets" {
			langs = append(langs, lang)
		}
	}
	return langs, nil
}

func collectGrepResults(source, pattern, path string, excludeExts []string) ([]GrepResult, error) {
	args := []string{"-r", "-n", "-I", "-E", "--color=never"}
	for _, ext := range excludeExts {
		if strings.HasPrefix(ext, ".") {
			ext = ext[1:]
		}
		args = append(args, fmt.Sprintf("--exclude=*.%s", ext))
	}
	args = append(args, pattern, path)

	cmd := exec.Command("grep", args...)
	outputBytes, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil // no matches
		}
		return nil, err
	}

	lines := strings.Split(string(outputBytes), "\n")
	var results []GrepResult

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		results = append(results, GrepResult{
			Source:  strings.ToUpper(source),
			Pattern: pattern,
			File:    parts[0],
			Line:    parts[1],
			Match:   parts[2],
		})
	}
	return results, nil
}

func printResultsPretty(results []GrepResult) {
	if len(results) == 0 {
		color.Green("\nNo matches found.\n")
		return
	}

	matchTag := color.New(color.FgGreen, color.Bold).SprintFunc()
	patternColor := color.New(color.FgHiMagenta, color.Bold).SprintFunc()
	pathColor := color.New(color.FgCyan).SprintFunc()
	highlightColor := color.New(color.FgHiRed, color.Bold).SprintFunc()
	faint := color.New(color.Faint).SprintFunc()

	trimAroundMatch := func(line, pattern string, contextLen int) string {
		re, err := regexp.Compile(pattern)
		if err != nil {
			if len(line) > 80 {
				return line[:77] + faint("…")
			}
			return line
		}
		loc := re.FindStringIndex(line)
		if loc == nil {
			if len(line) > 80 {
				return line[:77] + faint("…")
			}
			return line
		}
		start := loc[0] - contextLen
		if start < 0 {
			start = 0
		}
		end := loc[1] + contextLen
		if end > len(line) {
			end = len(line)
		}
		trimmed := line[start:end]
		if start > 0 {
			trimmed = faint("…") + trimmed
		}
		if end < len(line) {
			trimmed = trimmed + faint("…")
		}
		return trimmed
	}

	highlightMatches := func(text, pattern string) string {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return text
		}
		matches := re.FindAllStringIndex(text, -1)
		if matches == nil {
			return text
		}

		var result strings.Builder
		lastIndex := 0

		for _, match := range matches {
			start, end := match[0], match[1]
			result.WriteString(text[lastIndex:start])
			result.WriteString(highlightColor(text[start:end]))
			lastIndex = end
		}
		result.WriteString(text[lastIndex:])
		return result.String()
	}

	for _, r := range results {
		snippet := trimAroundMatch(r.Match, r.Pattern, 40)
		highlighted := highlightMatches(snippet, r.Pattern)

		fmt.Printf(
			"%s %s\n      %s %s\n\n",
			matchTag("["+r.Source+"]"),
			patternColor("Pattern: "+r.Pattern),
			pathColor(fmt.Sprintf("File: %s (Line %s)", r.File, r.Line)),
			highlighted,
		)
	}
}


func printBanner() {
	bold := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgHiCyan).SprintFunc()
	banner := `
 /$$$$$$$                                                   
| $$__  $$                                                  
| $$  \ $$  /$$$$$$   /$$$$$$   /$$$$$$  /$$   /$$ /$$   /$$
| $$$$$$$/ /$$__  $$ /$$__  $$ /$$__  $$|  $$ /$$/| $$  | $$
| $$__  $$| $$$$$$$$| $$  \ $$| $$$$$$$$ \  $$$$/ | $$  | $$
| $$  \ $$| $$_____/| $$  | $$| $$_____/  >$$  $$ | $$  | $$
| $$  | $$|  $$$$$$$|  $$$$$$$|  $$$$$$$ /$$/\  $$|  $$$$$$$
|__/  |__/ \_______/ \____  $$ \_______/|__/  \__/ \____  $$
                     /$$  \ $$                     /$$  | $$
                    |  $$$$$$/                    |  $$$$$$/
                     \______/                      \______/  By Abdullah Almutawa(@_d3caff)
`
	fmt.Println(cyan(banner))
	fmt.Println(bold(" Regexy - Dangerous Function & Secrets Scanner"))
	fmt.Println(strings.Repeat("─", 76))
}

func printUsage() {
	fmt.Println("\n Usage:")
	fmt.Println("  regexy -L <language|all> <path> [--secrets] [--exclude .ext1,.ext2]")
	fmt.Println("\n Examples:")
	fmt.Println("  regexy -L php ./myproject")
	fmt.Println("  regexy -L all ./codebase --secrets --exclude .js,.env,.css\n")
}

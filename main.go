package main

import (
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/net/html"
)

type Translation map[string]string

var variableRx = regexp.MustCompile("\\$[^$]+\\$")

// loadTranslation loads a <lang>.json into a map and returns it.
func loadTranslation(path string) Translation {
	bs, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("loadTranslation: %v: %v", path, err)
	}

	translation := Translation{}
	err = json.Unmarshal(bs, &translation)
	if err != nil {
		log.Fatalf("loadTranslation: %v: %v", path, err)
	}

	return translation
}

// checkTranslationsVariables checks for changed or missing variables.
// The reference is the english translations. If there are missing variables on either side,
// or the variables have been changed (possibly translated), report those as errors.
// The result is a map of translation[language] -> list of errors for that language.
// If the resulting map is empty, no errors were found.
func checkTranslationVariables(translations map[string]Translation) map[string][]string {
	result := make(map[string][]string)

	for enKey, enString := range translations["en"] {
		enMatches := variableRx.FindAllString(enString, -1)
		slices.Sort(enMatches)
		// Care about empty enMatches. That might mean that there are still variables
		// in the translation, but not in the original!
		for lang, translation := range translations {
			// Skip comparing english to english, and also missing translation strings.
			if lang == "en" || translation[enKey] == "" {
				continue
			}
			langMatches := variableRx.FindAllString(translation[enKey], -1)
			slices.Sort(langMatches)
			if slices.Compare(enMatches, langMatches) != 0 {
				result[lang] = append(result[lang],
					fmt.Sprintf("mismatch in variables: %v ⇒ %v\n",
						enString, translation[enKey]))
			}
		}
	}
	return result
}

func errStartWithoutEnd(start string) string {
	return fmt.Sprintf("starting tag without ending tag: <%v>", start)
}

func errEndWithoutStart(end string) string {
	return fmt.Sprintf("ending tag without starting tag: </%v>", end)
}

func errStartEndMismatch(start, end string) string {
	return fmt.Sprintf("starting and ending tags don't match: <%v>, </%v>", start, end)
}

// checkHTML checks whether the HTML tags in input are well balanced.
// An empty list is returned in case of success, otherwise a list of errors.
// TODO: It might be a good idea to optionally check against a list of accepted tags.
func checkHTML(input string) (errs []string) {
	tokenizer := html.NewTokenizer(strings.NewReader(input))
	l := list.New()
Out:
	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			e := tokenizer.Err()
			if !errors.Is(e, io.EOF) {
				errs = append(errs, fmt.Sprintf("unknown tokenizer error: %v", e))
			}
			break Out
		case html.StartTagToken:
			name, _ := tokenizer.TagName()
			l.PushFront(string(name))
		case html.EndTagToken:
			endb, _ := tokenizer.TagName()
			end := string(endb)
			el := l.Front()
			if el == nil {
				errs = append(errs, errEndWithoutStart(end))
				break
			}
			start := el.Value.(string)
			l.Remove(el)
			if start != end {
				errs = append(errs, errStartEndMismatch(start, end))
			}
		}
	}
	for el := l.Front(); el != nil; el = el.Next() {
		errs = append(errs, errStartWithoutEnd(el.Value.(string)))
	}
	return errs
}

func checkTranslationHTML(translations map[string]Translation) map[string][]string {
	result := make(map[string][]string)
	for lang, translation := range translations {
		for _, translatedString := range translation {
			errs := checkHTML(translatedString)
			for _, err := range errs {
				result[lang] = append(result[lang], fmt.Sprintf("%v: %v", err, translatedString))
			}
		}
	}
	return result
}

func main() {
	rootDir := processArgs()
	translations := make(map[string]Translation)

	// Build the translation maps.
	filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		base := filepath.Base(path)
		match, err := filepath.Match("??.json", base)
		if !match {
			return nil
		}
		if err != nil {
			return err
		}
		base, _ = strings.CutSuffix(base, ".json")
		translations[base] = loadTranslation(path)

		return nil
	})

	// Run the checks.
	variableErrors := checkTranslationVariables(translations)
	htmlErrors := checkTranslationHTML(translations)
	for lang, _ := range translations {
		if len(variableErrors[lang]) > 0 || len(htmlErrors[lang]) > 0 {
			fmt.Fprintf(os.Stderr, "[%v]\n", lang)
			for _, error := range variableErrors[lang] {
				fmt.Fprintf(os.Stderr, "    %v\n", error)
			}
			for _, error := range htmlErrors[lang] {
				fmt.Fprintf(os.Stderr, "    %v\n", error)
			}
		}
	}

	if len(variableErrors) > 0 || len(htmlErrors) > 0 {
		os.Exit(1)
	}
}

func processArgs() string {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage:\n    %v <translation-root-dir>\n", os.Args[0])
		os.Exit(1)
	}

	rootDir := os.Args[1]
	file, err := os.Open(rootDir)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}
	if !info.IsDir() {
		log.Fatal("must exist and be a readable directory: ", rootDir)
	}

	return os.Args[1]
}

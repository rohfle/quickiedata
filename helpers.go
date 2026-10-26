package quickiedata

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

// entityIDLocal matches the local part of an entity id: an item, property,
// lexeme, MediaInfo, form or sense.
const entityIDLocal = `(?:[QPLM][1-9]\d*|L[1-9]\d*-[FS][1-9]\d*)`

// ValidSPARQLEntityID matches a bare entity id, the form the entity API takes.
var ValidSPARQLEntityID = regexp.MustCompile(`^` + entityIDLocal + `$`)

// sparqlEntityID matches an entity id as a SPARQL value may write it: bare, or
// with wd:, the only prefix that names an entity. wdt:, p:, ps: and the rest
// name predicates, so they are not accepted here.
var sparqlEntityID = regexp.MustCompile(`^(?:wd:)?(` + entityIDLocal + `)$`)

// IsEntityID checks if a string is a bare entity id, as the entity API takes it.
func IsEntityID(id string) bool {
	return ValidSPARQLEntityID.MatchString(id)
}

// ParseEntityID accepts an entity id written bare (Q42) or with the entity
// prefix (wd:Q42) and returns it in the canonical wd: form.
func ParseEntityID(id string) (WikidataID, error) {
	match := sparqlEntityID.FindStringSubmatch(id)
	if match == nil {
		return "", fmt.Errorf("invalid entity id %q, want Q42 or wd:Q42", id)
	}
	return WikidataID("wd:" + match[1]), nil
}

// ValidateEntityID checks if a string is a valid entity id
func ValidateEntityID(id string) error {
	if !IsEntityID(id) {
		return fmt.Errorf("invalid entity id '%s'", id)
	}
	return nil
}

// ValidateEntityIDs checks an array of entity ids are all valid
func ValidateEntityIDs(ids []string) error {
	for _, id := range ids {
		if err := ValidateEntityID(id); err != nil {
			return err
		}
	}
	return nil
}

// ValidateEntityType checks if entity type is valid
func ValidateEntityType(entityType string) error {
	switch entityType {
	case "item", "property", "lexeme", "form", "sense":
		return nil
	default:
		return fmt.Errorf("invalid entity type '%s'", entityType)
	}
}

func GetEntityIDFromNumericIDAndType(entityType string, numericID int64) (string, error) {
	prefix := ""
	switch entityType {
	case "item":
		prefix = "Q"
	case "property":
		prefix = "P"
	case "lexeme":
		prefix = "L"
	// form and sense cannot be converted as they have two number parts (eg form L<xxx>-F<yyy>)
	default:
		return "", fmt.Errorf("cannot convert entity type '%s' to entity id", entityType)
	}
	return fmt.Sprintf("%s%d", prefix, numericID), nil
}

func ConvertLanguage(language string) string {
	return strings.ToLower(strings.SplitN(language, "_", 2)[0])
}

func ConvertLanguages(languages []string) []string {
	var newLanguages []string
	for _, language := range languages {
		newLanguages = append(newLanguages, ConvertLanguage(language))
	}
	return newLanguages
}

// GetSitelinkURL gets the full sitelink url from site and title
func GetSitelinkURL(site string, title string) string {
	if site == "" || title == "" {
		return ""
	}

	title = url.QueryEscape(strings.ReplaceAll(title, " ", "_"))

	var specialSites = []string{
		"commonswiki",
		"metawiki",
		"specieswiki",
		"wikimaniawiki",
	}

	if slices.Contains(specialSites, site) {
		site = strings.TrimSuffix(site, "wiki")
		return fmt.Sprintf("https://%s.wikimedia.org/wiki/%s", site, title)
	} else if site == "mediawikiwiki" {
		return fmt.Sprintf("https://www.mediawiki.org/wiki/%s", title)
	} else if site == "wikidatawiki" {
		switch title[0] {
		case 'E':
			title = "EntitySchema:" + title
		case 'L':
			title = "Lexeme:" + strings.ReplaceAll(title, "-", "#")
		case 'P':
			title = "Property:" + title
		}
		return fmt.Sprintf("https://www.wikidata.org/wiki/%s", title)
	}

	bits := strings.SplitN(site, "wiki", 2)
	if len(bits) < 2 || bits[0] == "" {
		return ""
	}
	lang := strings.ReplaceAll(bits[0], "_", "-")
	project := "wiki" + bits[1]
	if strings.HasSuffix(project, "wiki") {
		project = "wikipedia"
	}

	return fmt.Sprintf("https://%s.%s.org/wiki/%s", lang, project, title)
}

func GetWikidataIDFromURL(url string) string {
	if url == "" {
		return ""
	}
	return strings.TrimPrefix(url, "http://www.wikidata.org/entity/")
}

func SplitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	var out []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

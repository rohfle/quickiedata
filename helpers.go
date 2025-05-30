package quickiedata

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var ValidSPARQLEntityID = regexp.MustCompile("^((Q|P|L|M)[1-9][0-9]*|L[1-9][0-9]*-(F|S)[1-9][0-9]*)$")

// IsEntityID checks if a string is a valid entity id
func IsEntityID(id string) bool {
	return ValidSPARQLEntityID.MatchString(id)
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
		"wikidatawiki",
		"wikimaniawiki",
	}

	if ValueInSlice(site, specialSites) {
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

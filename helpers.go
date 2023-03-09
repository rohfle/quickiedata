package quickiedata

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var ENTITY_ID_REGEXP = regexp.MustCompile("^((Q|P|L|M)[1-9][0-9]*|L[1-9][0-9]*-(F|S)[1-9][0-9]*)$")

func IsEntityID(id string) bool {
	return ENTITY_ID_REGEXP.MatchString(id)
}

func ValidateEntityIDs(ids []string) error {
	for _, eid := range ids {
		if !IsEntityID(eid) {
			return fmt.Errorf("invalid entity id '%s'", eid)
		}
	}
	return nil
}

func ValidateEntityType(entityType string) error {
	switch entityType {
	case "item", "property", "lexeme", "form", "sense":
		return nil
	default:
		return fmt.Errorf("invalid entity type '%s'", entityType)
	}
}

func ConvertLanguages(languages []string) []string {
	var newLanguages []string
	for _, language := range languages {
		language := strings.ToLower(strings.SplitN(language, "_", 2)[0])
		newLanguages = append(newLanguages, language)
	}
	return newLanguages
}

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
		site = strings.TrimPrefix(site, "wiki")
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

	var SITE_PARSER_REGEXP = regexp.MustCompile(`^(.+)(wiki.*)$`)

	match := SITE_PARSER_REGEXP.FindAllStringSubmatch(site, 1)
	if match == nil {
		return ""
	}

	lang := match[0][1]
	project := match[0][2]
	lang = strings.ReplaceAll(lang, "_", "-")
	if strings.HasSuffix(project, "wiki") {
		project = "wikipedia"
	}

	return fmt.Sprintf("https://%s.%s.org/wiki/%s", lang, project, title)
}

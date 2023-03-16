package quickiedata

import "encoding/json"

// This is a combined struct for all entity info types
// Use the Type field to work out which fields will have values
type EntityInfo struct {
	PageID    int64     `json:"pageid"`
	NS        int64     `json:"ns"`
	Title     string    `json:"title"`
	LastRevID int64     `json:"lastrevid"`
	Modified  string    `json:"modified"`
	Redirects *Redirect `json:"redirects,omitempty"`
	Type      string    `json:"type"`

	// Item fields
	ID           string               `json:"id"`
	Labels       map[string]*Term     `json:"labels"`
	Descriptions map[string]*Term     `json:"descriptions"`
	Aliases      map[string][]*Term   `json:"aliases"`
	Claims       map[string][]*Claim  `json:"claims"`
	Sitelinks    map[string]*Sitelink `json:"sitelinks"`

	// Property fields
	// ID (defined above)
	DataType DataType `json:"datatype"`
	// Labels (defined above)
	// Descriptions (defined above)
	// Aliases (defined above)
	// Claims (defined above)

	// Lexeme fields
	// ID (defined above)
	// DataType (defined above)
	LexicalCategory string           `json:"lexical-category"`
	Language        string           `json:"language"`
	Lemmas          map[string]*Term `json:"lemmas"`
	Forms           []*Form          `json:"forms"`
	Senses          []*Sense         `json:"senses"`

	// Form fields
	// ID (defined above)
	Representations     map[string]*Term `json:"representations"`
	GrammaticalFeatures []string         `json:"grammatical-features"`
	// Claims (defined above)

	// Sense fields
	// ID (defined above)
	Glosses map[string]*Term `json:"glosses"`
	// Claims (defined above)
}

type Form struct {
	ID                  string              `json:"id"`
	Representations     map[string]*Term    `json:"representations"`
	GrammaticalFeatures []string            `json:"grammaticalFeatures"`
	Claims              map[string][]*Claim `json:"claims"`
}

type Sense struct {
	ID      string              `json:"id"`
	Glosses map[string]*Term    `json:"glosses"`
	Claims  map[string][]*Claim `json:"claims"`
}

type SimpleProperty struct {
	DataType     DataType                  `json:"datatype,omitempty"`
	Labels       map[string]string         `json:"labels,omitempty"`
	Descriptions map[string]string         `json:"descriptions,omitempty"`
	Aliases      map[string][]string       `json:"aliases,omitempty"`
	Claims       map[string][]*SimpleClaim `json:"claims,omitempty"`
}

func (s *SimpleProperty) MarshalJSON() ([]byte, error) {
	type proxy SimpleProperty
	return json.Marshal(struct {
		proxy
		Type string `json:"type"`
	}{
		proxy: proxy(*s),
		Type:  "property",
	})
}

func (s *SimpleProperty) GetClaims(key string) []*SimpleClaim {
	if s == nil {
		return nil
	}

	claims, ok := s.Claims[key]
	if !ok {
		return nil
	}

	return claims
}

func (s *SimpleProperty) GetClaim(key string) *SimpleClaim {
	if s == nil {
		return nil
	}

	claims := s.GetClaims(key)
	if len(claims) == 0 {
		return nil
	}
	return claims[0]
}

type SimpleItem struct {
	Labels       map[string]string         `json:"labels,omitempty"`
	Descriptions map[string]string         `json:"descriptions,omitempty"`
	Aliases      map[string][]string       `json:"aliases,omitempty"`
	Claims       map[string][]*SimpleClaim `json:"claims,omitempty"`
	Sitelinks    map[string]string         `json:"sitelinks,omitempty"`
}

func (s *SimpleItem) MarshalJSON() ([]byte, error) {
	type proxy SimpleItem
	return json.Marshal(struct {
		proxy
		Type string `json:"type"`
	}{
		proxy: proxy(*s),
		Type:  "item",
	})
}

func (s *SimpleItem) GetClaims(key string) []*SimpleClaim {
	if s == nil {
		return nil
	}
	claims, ok := s.Claims[key]
	if !ok {
		return nil
	}
	return claims
}

func (s *SimpleItem) GetClaim(key string) *SimpleClaim {
	if s == nil {
		return nil
	}
	claims := s.GetClaims(key)
	if len(claims) == 0 {
		return nil
	}
	return claims[0]
}

func (s *SimpleItem) GetRelatedIDsFromClaims(claimIDs []string) []string {
	if s == nil || len(claimIDs) == 0 {
		return nil
	}

	var relatedIDs []string
	for _, claim := range claimIDs {
		for _, item := range s.GetClaims(claim) {
			wikidataID := item.ValueAsString()
			if wikidataID != nil {
				relatedIDs = append(relatedIDs, *wikidataID)
			}
		}
	}
	return relatedIDs
}

func (s *SimpleItem) GetQualifierIDsFromClaim(claimID string, qualifierID string) []string {
	if s == nil {
		return nil
	}

	var relatedIDs []string

	for _, qualifier := range s.GetClaim(claimID).GetQualifiers(qualifierID) {
		wikidataID := qualifier.ValueAsString()
		if wikidataID != nil {
			relatedIDs = append(relatedIDs, *wikidataID)
		}
	}
	return relatedIDs
}

type SimpleLexeme struct {
	Type            string            `json:"type"`
	LexicalCategory string            `json:"category,omitempty"`
	Language        string            `json:"language,omitempty"`
	Lemmas          map[string]string `json:"lemmas,omitempty"`
	Forms           []*SimpleForm     `json:"forms,omitempty"`
	Senses          []*SimpleSense    `json:"senses,omitempty"`
}

func (s *SimpleLexeme) MarshalJSON() ([]byte, error) {
	type proxy SimpleLexeme
	return json.Marshal(struct {
		proxy
		Type string `json:"type"`
	}{
		proxy: proxy(*s),
		Type:  "lexeme",
	})
}

type SimpleForm struct {
	Type                string                    `json:"type"`
	Representations     map[string]string         `json:"representations,omitempty"`
	GrammaticalFeatures []string                  `json:"features,omitempty"`
	Claims              map[string][]*SimpleClaim `json:"claims,omitempty"`
}

func (s *SimpleForm) MarshalJSON() ([]byte, error) {
	type proxy SimpleForm
	return json.Marshal(struct {
		proxy
		Type string `json:"type"`
	}{
		proxy: proxy(*s),
		Type:  "form",
	})
}

func (s *SimpleForm) GetClaims(key string) []*SimpleClaim {
	if s == nil {
		return nil
	}
	claims, ok := s.Claims[key]
	if !ok {
		return nil
	}
	return claims
}

func (s *SimpleForm) GetClaim(key string) *SimpleClaim {
	if s == nil {
		return nil
	}
	claims := s.GetClaims(key)
	if len(claims) == 0 {
		return nil
	}
	return claims[0]
}

type SimpleSense struct {
	Type    string                    `json:"type"`
	Glosses map[string]string         `json:"glosses,omitempty"`
	Claims  map[string][]*SimpleClaim `json:"claims,omitempty"`
}

func (s *SimpleSense) MarshalJSON() ([]byte, error) {
	type proxy SimpleSense
	return json.Marshal(struct {
		proxy
		Type string `json:"type"`
	}{
		proxy: proxy(*s),
		Type:  "sense",
	})
}

func (s *SimpleSense) GetClaims(key string) []*SimpleClaim {
	if s == nil {
		return nil
	}
	claims, ok := s.Claims[key]
	if !ok {
		return nil
	}
	return claims
}

func (s *SimpleSense) GetClaim(key string) *SimpleClaim {
	if s == nil {
		return nil
	}
	claims := s.GetClaims(key)
	if len(claims) == 0 {
		return nil
	}
	return claims[0]
}

package types

type Claim struct {
	ID              string             `json:"id"`
	MainSnak        *Snak              `json:"mainsnak"`
	Rank            Rank               `json:"rank"`
	Type            DataType           `json:"type"`
	Qualifiers      map[string][]*Snak `json:"qualifiers"`
	QualifiersOrder []string           `json:"qualifiersOrder"`
	References      []*Reference       `json:"references"`
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

type Sitelink struct {
	Site   string   `json:"site"`
	Title  string   `json:"title"`
	Badges []string `json:"badges,omitempty"`
	URL    string   `json:"url"`
}

type Term struct {
	Language string `json:"language"`
	Value    string `json:"value"`
}

type Reference struct {
	Hash       string             `json:"hash"`
	Snaks      map[string][]*Snak `json:"snaks"`
	SnaksOrder []string           `json:"snaks-order"`
}

type Redirect struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type ResponseError struct {
	Code  string `json:"code"`
	Info  string `json:"info"`
	Extra string `json:"extra"`
}

func (re *ResponseError) Error() string {
	return re.Info
}

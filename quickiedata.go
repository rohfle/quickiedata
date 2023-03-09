package quickiedata

import (
	"github.com/rohfle/quickiedata/helpers"
	"github.com/rohfle/quickiedata/types"
)

var LookupCommonCalendars = helpers.LookupCommonCalendars
var LookupCommonGlobes = helpers.LookupCommonGlobes
var LookupCommonUnits = helpers.LookupCommonUnits

var NewGetEntitiesOptions = types.NewGetEntitiesOptions
var NewSearchEntities = types.NewSearchEntitiesOptions
var NewGetSPARQLQueryOptions = types.NewGetSPARQLQueryOptions

func WikidataID(s string) helpers.WikidataID {
	return helpers.WikidataID(s)
}

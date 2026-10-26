# Changelog

Newest first. Versions follow [Semantic Versioning](https://semver.org/); this
library is pre-1.0, so a breaking change bumps the minor version.

## v0.2.0 — 2026-10-26

Requires Go 1.26 or later; `go.mod` moved up from `go 1.23.0`.

### Breaking

- `NewClient(userAgent, options)` replaces `NewClient(settings)` and returns an
  error. The user agent is required and must name the client with contact
  details, as the Wikimedia user agent policy asks.
- `SPARQLQuery`, `SPARQLQuerySimple` and `SPARQLQueryRaw` take the context and
  the query only. `GetSPARQLQueryOptions` and `NewSPARQLQueryOptions` are gone.
- `SPARQLQuery.Variables` is a `Variables` (`map[string][]SPARQLTerm`) with
  `Set` and `Add`, replacing `map[string]any`. Values are typed terms:
  `String`, `Text`, `Integer`, `Float`, `Decimal`, `Boolean`, `Coordinate`,
  `WikidataDateTime`, `WikidataDuration`, `WikidataID` and `Predicate`, with
  `Strings`, `IDs` and `Terms` for lists. One term renders as a `BIND`, several
  as a `VALUES` clause.
- `WikidataID` renders as the full entity IRI
  (`<http://www.wikidata.org/entity/Q42>`) instead of a `wd:Q42` QName.
- `SPARQLQuery.Offset` and `Limit` are appended only when set; the template is
  no longer sniffed for a trailing clause. A template that carries its own
  `OFFSET` or `LIMIT` now gets a second one, which the endpoint rejects.
- Variable statements are injected per scope: the whole block goes into the
  outer group pattern, and each variable's statement is repeated inside every
  subquery that uses it. A query that sets variables but has no group pattern
  returns `ErrNoInjectionPoint` instead of dropping them, and an implicit group
  (`SELECT ?x { ... }`) is now found.
- `GetEntitySimpleResponse.GetEntityAsItem`, `GetEntityAsLexeme` and
  `GetEntityAsProperty` lose their unused `key` parameter.
- `ValueInSlice` is removed.

### Added

- `Version()` reports the release version from build info.
- `SPARQLCompatibilityIssues` lists constructs that WDQS v1 accepts and v2 does
  not, reported by `quickiedata-cli check-query`.
- `ParseEntityID` and `ParsePredicate`.
- `quickiedata-cli`: `--offset` and `--limit` apply only when set, `--debug`
  logs the rendered query and the scanner view to stderr, `[a,b]` binds a list,
  and each `key=value` is classified as an entity id, a predicate or a string.

## v0.1.0 — 2025-05-31

Initial release.

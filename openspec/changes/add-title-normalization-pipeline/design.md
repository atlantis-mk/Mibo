## Context

The scanner currently derives catalog titles in `internal/library/scan_classify.go` using `cleanTitle()`, while TMDB search query construction in `internal/metadata/service_matcher.go` uses a separate `cleanSearchTitle()` implementation. Both remove some release noise, but their rule sets are duplicated and incomplete for website watermarks and release-site tags. The scanner already preserves `OriginalTitle`, year, and scanner metadata evidence, and catalog scan updates already preserve descriptive fields for matched, reviewed, locked, or manual items.

Resolution, codec, audio, and subtitle information is already probed separately through ffprobe and stored on inventory/media stream records. Filename-derived quality tokens should therefore be treated as title noise, not as authoritative technical metadata.

## Goals / Non-Goals

**Goals:**

- Provide one shared backend normalization path for scanner title generation and metadata search query cleanup.
- Remove common filename noise, including website watermarks, release-site tags, quality labels, HDR labels, codecs, source labels, platform labels, audio/subtitle labels, years, and trailing release groups.
- Extract year values into structured output while removing them from normalized title text.
- Preserve original filename-derived titles and add normalization evidence for removed tokens and reasons.
- Keep existing catalog governance protections for already matched or manually controlled descriptive fields.
- Add test coverage with realistic movie, TV, Chinese, URL watermark, and release-group filename examples.

**Non-Goals:**

- This change does not add a frontend configuration UI for custom blocked tokens.
- This change does not infer authoritative resolution, codec, audio, or subtitle metadata from filenames.
- This change does not replace TMDB matching or manual catalog governance workflows.
- This change does not rename existing catalog items that are protected by matched, reviewed, locked, or manual governance state unless existing scan update rules already permit it.

## Decisions

### Shared `titleclean` package

Create an internal package such as `internal/titleclean` with a small structured API:

```go
type NormalizeInput struct {
    RawTitle string
}

type NormalizeResult struct {
    Title                string
    Year                 *int
    RemovedTokens        []RemovedToken
    NormalizationVersion string
}

type RemovedToken struct {
    Value  string
    Reason string
}
```

The package should own filename token normalization and be called by both scanner classification and metadata search query generation. Keeping it internal avoids creating an external API commitment while eliminating duplicated rule drift.

Alternative considered: extend both existing regex sets independently. This is faster initially but keeps scanner and matcher behavior inconsistent and makes future rule changes riskier.

### Structured evidence rather than silent cleanup

Normalization should return removed tokens with reason labels such as `year`, `website`, `quality`, `hdr`, `video_codec`, `source`, `platform`, `audio`, `subtitle`, and `release_group`. Scanner metadata evidence should include these values plus a normalization version.

Alternative considered: only return the cleaned string. This would be simpler, but it makes false positives difficult to diagnose and prevents support/debug workflows from seeing why a title changed.

### Technical metadata remains probe-owned

Quality, codec, audio, and subtitle filename tokens are removed from title candidates but are not used to populate authoritative technical fields. The existing ffprobe/probe flow remains the source of truth for width, height, codecs, stream languages, and layouts.

Alternative considered: derive fallback technical metadata from filename tokens. That would add ambiguity and could conflict with probed data, especially for mislabeled releases.

### Conservative fallback behavior

If normalization produces an empty or unusably short title, callers must fall back to the trimmed original title. Chinese titles must not be treated as release groups by ASCII uppercase heuristics. Existing governance preservation stays unchanged so trusted metadata is not overwritten by rescan noise.

Alternative considered: aggressively strip all recognized-looking bracketed tokens. That would remove more watermarks but risks damaging legitimate titles, especially non-English titles or titles with bracketed edition names.

### Optional configuration seam, no UI in first implementation

The normalizer may expose a config type for future extra blocked tokens or regexes, but the first implementation should use built-in defaults only unless an existing backend settings path can be reused without widening scope.

Alternative considered: build full UI-driven custom rules immediately. That is useful long term but adds frontend, persistence, validation, and support complexity beyond the current need.

## Risks / Trade-offs

- False-positive cleanup can damage legitimate titles → Preserve `OriginalTitle`, record removed-token evidence, add conservative fallbacks, and cover multilingual examples in tests.
- Regex growth can become hard to maintain → Centralize rules in one package with reason categories and table-driven tests.
- Search behavior may change for existing noisy items → Use normalized query variants alongside existing title sources where useful, and keep fallback variants from original/path-derived titles.
- Website watermark patterns vary widely → Cover common URL/domain forms now and leave a future configuration seam for site-specific tokens.
- Existing matched metadata could be overwritten by rescans → Reuse existing catalog scan metadata override protections and add regression coverage if touched.

## Migration Plan

No database migration is expected. Existing catalog rows remain valid. Future scans will produce improved scanner-derived titles for unprotected items and updated scanner metadata evidence. Matched, reviewed, locked, or manually edited items continue to preserve descriptive fields according to current governance rules.

Rollback is code-level: revert the shared normalizer integration. Because raw original titles and source paths are preserved, no irreversible data transformation is introduced.

## Open Questions

- Which exact site-specific watermark tokens should be included in the initial built-in list beyond generic URL/domain matching?
- Should normalization evidence include token character offsets, or are value/reason pairs sufficient for current debugging needs?

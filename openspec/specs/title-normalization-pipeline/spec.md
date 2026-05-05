## Requirements

### Requirement: Shared title normalization
The system SHALL use a shared backend title normalization capability for scanner-derived titles and metadata search query cleanup so equivalent filename noise is handled consistently across catalog ingestion and matching.

#### Scenario: Scanner and matcher use equivalent cleanup
- **WHEN** a filename-derived title contains release noise such as `1080p`, `WEB-DL`, `x265`, and a website domain
- **THEN** scanner title generation and metadata search query cleanup both remove those noise tokens from title candidates

### Requirement: Website and release-site watermark removal
The system SHALL remove common website and release-site watermark tokens from normalized title candidates, including bracketed URL/domain forms and standalone domain-like tokens.

#### Scenario: Bracketed website watermark
- **WHEN** the scanner normalizes `[www.example.com]Some.Movie.2023.HD1080P`
- **THEN** the normalized title candidate is `Some Movie` and the extracted year is `2023`

#### Scenario: Embedded release-site domain
- **WHEN** the scanner normalizes `Show.Name[www.4KHDR.CN].S02E01.2025.2160p.WEB-DL.H264`
- **THEN** the website token is removed from title candidates without preventing season and episode detection

### Requirement: Structured year extraction
The system SHALL extract supported release years from filename-derived titles into structured year output and remove those year tokens from normalized title text.

#### Scenario: Movie year extraction
- **WHEN** the scanner normalizes `Movie.Name.2024.1080p.WEB-DL.x265-GROUP`
- **THEN** the normalized title candidate is `Movie Name` and the extracted year is `2024`

### Requirement: Technical filename noise removal
The system SHALL remove technical filename noise from normalized title candidates after preserving structured filename signals, including quality labels, HDR labels, video codecs, source labels, platform labels, audio markers, subtitle markers, language markers, and trailing release groups.

#### Scenario: Dense movie release name
- **WHEN** the scanner normalizes `Dune.Part.Two.2024.2160p.UHD.BluRay.REMUX.HEVC.TrueHD.Atmos-GROUP`
- **THEN** the normalized title candidate is `Dune Part Two` and technical tokens are not retained in the title

#### Scenario: TV release name
- **WHEN** the scanner classifies `Show.Name.S01E02.1080p.NF.WEB-DL.DDP5.1.Atmos.x264-GROUP`
- **THEN** the item is classified as episode `Show Name S01E02` without retaining technical release tokens in the series title

#### Scenario: Technical tokens are preserved as signals before removal
- **WHEN** title normalization removes `2160p`, `BluRay`, `HEVC`, `TrueHD`, `Atmos`, or `DDP5.1` from a filename-derived title
- **THEN** those tokens SHALL remain available as filename-derived release hints and removed-token evidence for classifier and scanner decision output

### Requirement: Technical metadata source of truth
The system SHALL treat filename quality, codec, audio, subtitle, and source tokens as title noise and filename-derived hints only, and SHALL NOT populate authoritative resolution, codec, audio, or subtitle fields from filename tokens.

#### Scenario: Resolution token in filename
- **WHEN** a filename contains `2160p` but ffprobe reports a different width and height
- **THEN** catalog and playback technical metadata use the ffprobe-derived stream data, not the filename token

#### Scenario: Audio token in filename
- **WHEN** a filename contains `DDP5.1` or `TrueHD.Atmos.7.1`
- **THEN** the token SHALL be available as filename-derived audio evidence but SHALL NOT populate authoritative audio stream fields without media probing

### Requirement: Normalization evidence preservation
The system SHALL preserve the original title and record scanner normalization evidence that includes the normalized title, normalization version, removed tokens with reason labels, and any filename signal references needed to explain why tokens were removed from title candidates.

#### Scenario: Removed token evidence
- **WHEN** scanner normalization removes `2024`, `2160p`, `WEB-DL`, `x265`, and `www.example.com`
- **THEN** scanner metadata evidence records each removed token with a reason label and preserves the original title separately

#### Scenario: Removed token is also classification evidence
- **WHEN** scanner normalization removes `DDP5.1` from a filename-derived title
- **THEN** the evidence SHALL show that the token was removed from the title as audio release noise and can suppress weak numeric episode inference

### Requirement: Conservative fallback and governance protection
The system SHALL fall back to the original trimmed title when normalization produces an empty or unusably short title, and SHALL preserve existing descriptive fields for catalog items protected by matched, needs-review, locked, or manual governance state.

#### Scenario: Empty normalization result
- **WHEN** all tokens in a filename-derived title are removed by normalization
- **THEN** the system uses the original trimmed title as the title candidate rather than writing an empty title

#### Scenario: Matched item rescan
- **WHEN** a catalog item with matched governance state is rescanned from a noisy filename
- **THEN** scanner normalization evidence may be refreshed but the existing protected title, original title, and year fields are not overwritten by scanner-derived values

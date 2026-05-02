## MODIFIED Requirements

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

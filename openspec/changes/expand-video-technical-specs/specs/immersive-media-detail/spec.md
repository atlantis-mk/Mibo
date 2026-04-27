## ADDED Requirements

### Requirement: Detail page renders MediaInfo-style video specifications
The system SHALL render available video stream technical attributes on the immersive media detail page using a MediaInfo-style label/value presentation.

#### Scenario: Primary asset has detailed video technical attributes
- **WHEN** a user opens a catalog-backed media detail page whose primary asset has a video stream with detailed technical attributes
- **THEN** the video information card MUST display fields such as title, codec, profile, level, resolution, aspect ratio, interlace state, frame rate, bitrate, color space, bit depth, pixel format, and reference frames when those values are available

#### Scenario: Multiple video streams exist
- **WHEN** a primary asset has multiple video streams
- **THEN** the video information card MUST distinguish each stream and render the available technical specification fields for each stream

#### Scenario: Video technical attributes are incomplete
- **WHEN** a primary asset has a video stream with only compact metadata such as codec and dimensions
- **THEN** the video information card MUST still render the available compact values and avoid empty rows for unavailable detailed fields

### Requirement: Detail page keeps non-video technical summaries available
The system SHALL preserve existing audio, subtitle, and file technical summaries while expanding the video stream display.

#### Scenario: User views media information after the video display changes
- **WHEN** the media detail page renders expanded video technical specifications
- **THEN** the page MUST still show available audio tracks, subtitle tracks, and file summary information in the media information section

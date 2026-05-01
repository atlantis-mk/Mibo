# scan-phase-artwork-preselection Specification

## Purpose
TBD - created by archiving change frontload-scan-artwork. Update Purpose after archive.
## Requirements
### Requirement: Preselect deterministic artwork during scan
The scanner SHALL write selected catalog images during the initial scan when deterministic low-cost artwork candidates are available and no authoritative selected image already exists for the same catalog item and image type.

#### Scenario: Sibling poster applied during first scan
- **WHEN** a scanned movie folder contains a video file and an unambiguous sibling poster candidate such as `poster.jpg`, `cover.jpg`, or `folder.jpg`
- **THEN** the scan result MUST create or update the catalog item with a selected `poster` image before metadata matching or probing jobs complete

#### Scenario: Sibling backdrop applied during first scan
- **WHEN** a scanned movie or series folder contains an unambiguous sibling backdrop candidate such as `backdrop.jpg` or `fanart.jpg`
- **THEN** the scan result MUST create or update the catalog item with a selected `backdrop` image before metadata matching or probing jobs complete

#### Scenario: Existing authoritative image is preserved
- **WHEN** a catalog item already has a selected image from manual governance or remote metadata for the same image type
- **THEN** the scanner MUST NOT replace that selected image with scan-phase artwork

### Requirement: Use provider thumbnails as provisional artwork fallback
The scanner SHALL use provider-supplied thumbnail URLs as provisional selected artwork when no sibling artwork or authoritative selected image is available for the target image type.

#### Scenario: Provider thumbnail fills missing movie poster
- **WHEN** a scanned movie has no selected poster and the storage object exposes a thumbnail URL
- **THEN** the scan result MUST create or update a selected `poster` image using the thumbnail URL as provisional artwork

#### Scenario: Provider thumbnail fills missing episode still
- **WHEN** a scanned episode has no selected still or backdrop and the storage object exposes a thumbnail URL
- **THEN** the scan result MUST create or update a selected `still` image using the thumbnail URL as provisional artwork

#### Scenario: Curated sibling artwork wins over provider thumbnail
- **WHEN** both an unambiguous sibling poster file and a provider thumbnail URL are available for the same scanned item
- **THEN** the scanner MUST select the sibling poster for the poster slot instead of the provider thumbnail

### Requirement: Keep expensive enrichment asynchronous
The scan phase SHALL NOT block on media probing, frame extraction, remote metadata lookup, remote artwork download, or remote artwork proxy caching to produce initial selected images.

#### Scenario: Remote metadata provider is slow
- **WHEN** TMDB or MetaTube is configured but slow or unavailable during a first scan
- **THEN** the scanner MUST still write catalog items and any deterministic scan-phase artwork without waiting for the remote metadata provider

#### Scenario: ffmpeg extraction remains delayed
- **WHEN** no scan-phase artwork candidate is available for a scanned file
- **THEN** the scanner MUST leave artwork enrichment to asynchronous probe or metadata jobs instead of running ffmpeg inline

### Requirement: Later enrichment can replace provisional scan artwork
Asynchronous metadata and probe enrichment SHALL be able to replace scan-phase provisional artwork with higher-authority selected images while preserving governed or manual selections.

#### Scenario: Remote poster replaces provisional thumbnail
- **WHEN** a metadata matching job later finds a remote poster for an item whose selected poster came from a provider thumbnail
- **THEN** the metadata job MUST be allowed to select the remote poster according to existing image selection rules

#### Scenario: Manual selection remains protected
- **WHEN** a user has manually selected artwork for a catalog item
- **THEN** neither scan-phase artwork nor asynchronous fallback extraction MUST overwrite that manual selection without an explicit governance action


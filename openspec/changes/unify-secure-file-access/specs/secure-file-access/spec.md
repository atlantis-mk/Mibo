## ADDED Requirements

### Requirement: External file access SHALL use short-lived signed grants
The system SHALL issue short-lived, signed access URLs for all external playback and artwork reads instead of exposing permanent resource URLs. Each signed grant MUST encode the access subject, purpose, and expiration time, and the server MUST reject expired or invalid grants.

#### Scenario: Playback URL expires after issuance
- **WHEN** a client requests playback access for an inventory file and later reuses the issued playback URL after its expiration time
- **THEN** the server MUST reject the request as expired

#### Scenario: Artwork URL is purpose-bound
- **WHEN** a client presents a signed artwork access URL to a playback access endpoint
- **THEN** the server MUST reject the request because the grant purpose does not match

### Requirement: Runtime access SHALL be derived from stable locators
The system SHALL treat provider identity and storage path as the stable source of truth for file access. External-facing access URLs, remote provider URLs, and local filesystem paths MUST NOT be used as durable identifiers for future reads.

#### Scenario: Provider URL is not persisted as the canonical locator
- **WHEN** the system needs to read an OpenList-backed file after a previously observed provider URL has expired
- **THEN** it MUST resolve access again from the stored provider and storage path rather than trusting the stale URL

#### Scenario: Local file access hides absolute filesystem paths from clients
- **WHEN** a client requests playback or artwork for a local inventory file
- **THEN** the client-visible response MUST use a signed Mibo access URL instead of exposing the absolute filesystem path

### Requirement: Access verification SHALL support local and remote serving modes
After a signed grant is verified, the system SHALL support serving the underlying content from a verified local filesystem path or from a runtime-resolved remote provider URL without changing the client contract.

#### Scenario: Verified local playback grant serves a local file
- **WHEN** a signed playback grant targets a local inventory file
- **THEN** the server MUST verify the grant and serve the file contents from the local filesystem

#### Scenario: Verified OpenList playback grant resolves a current provider URL
- **WHEN** a signed playback grant targets an OpenList inventory file
- **THEN** the server MUST verify the grant and resolve a current provider-backed access path before returning content

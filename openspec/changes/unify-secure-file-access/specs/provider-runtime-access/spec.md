## ADDED Requirements

### Requirement: OpenList runtime access SHALL be refreshed on demand
For OpenList-backed resources, the system SHALL obtain access at read time from the current provider state using the stable provider locator. Runtime access results MUST be treated as volatile and MUST NOT be assumed reusable across later requests.

#### Scenario: OpenList playback grant refreshes a provider link
- **WHEN** the system handles a signed playback request for an OpenList inventory file
- **THEN** it MUST resolve a current provider access path from the OpenList provider at request time

#### Scenario: OpenList metadata read does not reuse stale access data
- **WHEN** sidecar hydration or probe needs to read an OpenList-backed object after a previously returned provider URL has expired
- **THEN** the system MUST request fresh runtime access from the provider using the stored locator

### Requirement: Local runtime access SHALL be authorized like remote access
For local-backed resources, the system SHALL use the same signed access verification flow as remote providers before serving content, even though the bytes are read directly from disk.

#### Scenario: Local artwork request requires a valid signed grant
- **WHEN** a client requests a local metadata image through the access endpoint without a valid signed grant
- **THEN** the server MUST reject the request

#### Scenario: Local playback request serves bytes after verification
- **WHEN** a client requests a local playback access URL with a valid signed grant
- **THEN** the server MUST verify the grant and serve the file from disk without exposing the path to the client

### Requirement: Internal metadata and probe consumers SHALL use purpose-specific runtime access
Internal consumers such as sidecar hydration and ffprobe SHALL request runtime access through the shared provider access service instead of implementing provider-specific `Get`/`Link` fallback logic in each caller.

#### Scenario: Probe requests runtime access through the shared service
- **WHEN** ffprobe needs to inspect an inventory file
- **THEN** the probe flow MUST obtain a purpose-specific runtime access grant from the shared service rather than calling provider fallback logic directly

#### Scenario: Sidecar hydration requests runtime access through the shared service
- **WHEN** sidecar hydration needs to read metadata content for a provider-backed sidecar file
- **THEN** the hydration flow MUST obtain a purpose-specific runtime access grant from the shared service rather than implementing provider-specific URL fallback logic

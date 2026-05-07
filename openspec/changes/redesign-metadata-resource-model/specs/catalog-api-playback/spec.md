## MODIFIED Requirements

### Requirement: Playback selects resources for metadata
The playback API SHALL resolve playback from a metadata identity and an optional resource identifier.

#### Scenario: Playback with explicit resource
- **WHEN** a client requests playback for a metadata identity and resource ID
- **THEN** the system validates that the resource is linked to the metadata identity and is accessible before returning a playable source

#### Scenario: Playback without explicit resource
- **WHEN** a client requests playback for a metadata identity without a resource ID
- **THEN** the system selects an available resource using user preference, recent progress, primary role, and quality policy

### Requirement: Playback supports multi-episode resources
The playback API SHALL support resources linked to multiple episode metadata identities.

#### Scenario: Multi-episode playback target
- **WHEN** a resource is linked to multiple episodes and the client requests one episode metadata identity
- **THEN** the playback response identifies the requested episode segment and resource source file

### Requirement: Playback honors library context
The playback API SHALL use library context for policy decisions when provided.

#### Scenario: Library subtitle policy
- **WHEN** playback is requested with a library context
- **THEN** subtitle selection uses that library's subtitle policy for the selected resource

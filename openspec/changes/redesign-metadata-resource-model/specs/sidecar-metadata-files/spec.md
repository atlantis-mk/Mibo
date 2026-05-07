## MODIFIED Requirements

### Requirement: Sidecar evidence attaches to resources
Sidecar metadata parsing SHALL attach evidence to resources and files before it enriches metadata identities.

#### Scenario: Sidecar with external ID
- **WHEN** a sidecar file provides a provider external ID for a resource
- **THEN** the system uses that evidence to link or create the corresponding global metadata identity

#### Scenario: Sidecar with local fields only
- **WHEN** a sidecar file provides title or overview without an external ID
- **THEN** the system records a local metadata source with evidence linked to the resource and applies fields according to governance policy

### Requirement: Sidecar does not imply library ownership
Sidecar metadata SHALL NOT make a metadata identity owned by the library that scanned the sidecar.

#### Scenario: Sidecar scanned from one library
- **WHEN** a library scan reads a sidecar and enriches a metadata identity
- **THEN** the resulting metadata identity remains global and can be linked by resources from other libraries

## ADDED Requirements

### Requirement: Governance workspaces use metadata resources
The system SHALL expose governance workspaces in terms of metadata items, resources, resource files, resource links, projections, and metadata operation evidence rather than catalog items and assets.

#### Scenario: Client opens governance workspace
- **WHEN** a client requests governance details for a metadata item
- **THEN** the response MUST include field states, source evidence, selected images, resource links, resource files, projection visibility, and correction actions without requiring asset-link context

### Requirement: Manual restructure corrections use resources
The system SHALL apply movie-version, independent-movie, episode-sequence, merge, split, relink, unlink, and role corrections through resource metadata links and metadata projections.

#### Scenario: User restructures movie versions
- **WHEN** a user confirms that sibling playable resources are versions of one movie
- **THEN** governance MUST link those resources to the target metadata item with version roles and rebuild affected projections

#### Scenario: User restructures independent movies
- **WHEN** a user confirms that sibling playable resources are independent movies
- **THEN** governance MUST create or select separate metadata identities, link each resource to the correct identity, and avoid asset-item writes

#### Scenario: User restructures episode sequence
- **WHEN** a user confirms episode numbering for a set of resources
- **THEN** governance MUST link resources to series/season/episode metadata identities with segment indexes when needed

## REMOVED Requirements

### Requirement: Governance exposes image and asset relationship management
**Reason**: Asset relationship management is replaced by resource relationship governance.
**Migration**: Use resource list, resource link, resource update, metadata merge/split, and projection visibility governance endpoints.

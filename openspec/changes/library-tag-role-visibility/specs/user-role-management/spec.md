## ADDED Requirements

### Requirement: Roles can carry library visibility rules
The system MUST allow role definitions to store library access tag rules in addition to their existing management-access semantics.

#### Scenario: Add allow rules to a role
- **WHEN** an administrator updates a role with one or more allowed library access tags
- **THEN** the system SHALL persist those allow rules as part of the role definition

#### Scenario: Add deny rules to a role
- **WHEN** an administrator updates a role with one or more denied library access tags
- **THEN** the system SHALL persist those deny rules as part of the role definition

### Requirement: User visibility scope is derived from all assigned roles
The system MUST compute a user's library visibility scope from the full set of roles assigned to that user rather than from only a single primary role field.

#### Scenario: Multiple roles expand allowed scope
- **WHEN** a user has multiple assigned roles with different allowed library access tags
- **THEN** the system SHALL evaluate the union of those allow rules before applying deny precedence

#### Scenario: Multiple roles preserve deny precedence
- **WHEN** one assigned role allows a library access tag and another assigned role denies the same tag
- **THEN** the system SHALL treat the deny rule as authoritative for that user

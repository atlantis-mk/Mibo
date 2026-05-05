## ADDED Requirements

### Requirement: Library detail displays organizing summaries
The system SHALL display condition-derived organizing summaries for library browse results that are discovered, organizing, partially ready, ready, failed, or review-required.

#### Scenario: Discovered file appears before final catalog materialization
- **WHEN** a library contains an inventory-backed discovered video whose ingest conditions are not fully materialized
- **THEN** the library detail view MUST render it as a media card with an organizing state and concise progress copy

#### Scenario: Catalog-backed item still has background work
- **WHEN** a catalog-backed browse result has pending or failed probe, metadata, or projection conditions
- **THEN** the library detail view MUST surface an appropriate organizing, partial-ready, or review-required badge without duplicating the result as a separate discovered card

#### Scenario: Organizing state changes after reconciliation
- **WHEN** reconciliation updates an item's organizing summary from organizing to ready or review-required
- **THEN** the library detail view MUST update the card state using normal data refresh behavior without requiring a full page reload

### Requirement: Organizing cards expose safe actions only
The system SHALL limit user actions on organizing or review-required cards to actions that have a valid target identity and clear behavior.

#### Scenario: Card has no final catalog item yet
- **WHEN** a discovered organizing card is anchored only to an inventory file
- **THEN** the library detail view MUST hide or disable final catalog-only actions such as favorite, metadata edit, and detailed catalog navigation
- **AND** it MAY keep safe actions such as playback when a playable source exists

#### Scenario: Card requires metadata review
- **WHEN** a catalog-backed card has a review-required metadata or classification condition
- **THEN** the library detail view MUST provide a route or affordance to the existing manual match/governance flow when the user has permission

## ADDED Requirements

### Requirement: User-Scoped Favorites use metadata identities
The system SHALL allow an authenticated user to mark metadata items as favorites and persist those favorites per user without `UserItemData` fallback.

#### Scenario: Add favorite
- **WHEN** an authenticated user favorites a metadata item
- **THEN** the item is added to that user's metadata favorites and remains favorited after page reload or a new session

#### Scenario: Remove favorite
- **WHEN** an authenticated user removes a favorite from a metadata item
- **THEN** the metadata favorite is removed without affecting other users or resource progress

### Requirement: Favorites resolve library and resource context at read time
The system SHALL resolve favorites display rows from metadata item data plus current library projection and resource availability.

#### Scenario: Favorite exists in multiple libraries
- **WHEN** a metadata item is visible in more than one library
- **THEN** the favorites response MUST choose a deterministic display context and MUST NOT duplicate the favorite solely because multiple library projections exist

#### Scenario: Favorited metadata has no available resources
- **WHEN** a favorited metadata item has no available resources in the active context
- **THEN** favorites MUST still show the metadata item with unavailable state rather than dropping the favorite

## REMOVED Requirements

### Requirement: User-Scoped Favorites
**Reason**: The prior requirement used catalog item identity; favorites now use metadata item identity.
**Migration**: Use metadata favorite state and metadata item IDs for add/remove/list behavior.

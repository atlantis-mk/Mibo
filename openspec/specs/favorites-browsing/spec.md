# favorites-browsing Specification

## Purpose
Define user-scoped favorites behavior and the favorites browsing surface.

## Requirements

### Requirement: User-Scoped Favorites
The system SHALL allow an authenticated user to mark catalog items as favorites and persist those favorites per user.

#### Scenario: Add favorite
- **WHEN** an authenticated user favorites a catalog item
- **THEN** the item is added to that user's favorites and remains favorited after page reload or a new session

#### Scenario: Remove favorite
- **WHEN** an authenticated user removes a favorite from a catalog item
- **THEN** the item is removed from that user's favorites without affecting other users

### Requirement: Favorites Browsing Surface
The system SHALL provide a favorites browsing surface reachable from the homepage top navigation and the app sidebar.

#### Scenario: Open favorites
- **WHEN** the user selects Favorites from the top navigation or sidebar
- **THEN** the app displays the user's favorite catalog items using the same poster-card behavior as other browsing surfaces

#### Scenario: Empty favorites
- **WHEN** the user opens Favorites and has not favorited any items
- **THEN** the app displays an empty state that explains how to add favorites

### Requirement: Favorite State Visibility
The system SHALL show whether a catalog item is currently favorited on item detail and on card surfaces that expose favorite actions.

#### Scenario: Favorite state on detail
- **WHEN** a user opens a favorited item's detail page
- **THEN** the detail page indicates that the item is already in favorites

#### Scenario: Favorite state after mutation
- **WHEN** a user adds or removes a favorite from a card or detail page
- **THEN** the visible favorite state updates after the mutation succeeds

### Requirement: Favorite Access Control
The system MUST require authentication for listing, adding, and removing favorites.

#### Scenario: Unauthenticated favorite request
- **WHEN** a request without a valid authenticated session attempts to list or mutate favorites
- **THEN** the system rejects the request using the app's normal authentication error behavior

#### Scenario: User isolation
- **WHEN** two different users favorite different catalog items
- **THEN** each user only sees and mutates their own favorites

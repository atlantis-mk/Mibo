## ADDED Requirements

### Requirement: Person cards open a dedicated person detail route
The system SHALL let authenticated users open a dedicated person detail page from catalog media detail cast and director cards.

#### Scenario: Open person detail from media detail
- **WHEN** a user activates a cast or director card on a catalog media detail page and that person has a valid catalog person ID
- **THEN** the app MUST navigate to that person's dedicated detail route instead of treating the card as static content

#### Scenario: Person card focus behavior
- **WHEN** a keyboard user tabs to a cast or director card that links to a person detail page
- **THEN** the card MUST expose a visible focus state and an accessible name based on the person's display name

### Requirement: Person detail hero prioritizes identity and readable facts
The system SHALL render a person-first hero that emphasizes portrait, name, biography, and available life facts before related works or external links.

#### Scenario: Person has rich metadata
- **WHEN** a person detail response includes portrait, biography, birthday, place of birth, and a related-work backdrop candidate
- **THEN** the page MUST show the portrait as the primary visual, the person's name as the primary heading, the biography in the hero information area, and the related-work backdrop as immersive background artwork

#### Scenario: Optional facts are missing
- **WHEN** a person detail response lacks some optional fields such as birthday, age, place of birth, biography, or backdrop artwork
- **THEN** the page MUST omit empty fact rows and fall back to a stable visual placeholder without showing broken images or placeholder text that implies unknown data is an error

### Requirement: Person detail page exposes local related works for navigation
The system SHALL show the person's related catalog titles as a poster-card browsing section that routes users back into local media detail pages.

#### Scenario: Related works exist
- **WHEN** the person detail response includes related catalog items
- **THEN** the page MUST render a works section with poster cards that include at least artwork, title, and year or year range when available

#### Scenario: User opens a related work
- **WHEN** a user activates a related work card from the person detail page
- **THEN** the app MUST navigate to the corresponding local media detail route for that catalog item

#### Scenario: No related works exist
- **WHEN** a person detail response has no related catalog items
- **THEN** the page MUST omit the poster shelf and instead show a concise empty state explaining that no local titles are currently linked to this person

### Requirement: Person detail page exposes external reference links when known
The system SHALL show outbound person reference links for supported metadata providers when those identities are available.

#### Scenario: Supported external identities exist
- **WHEN** a person detail response includes TMDB or IMDb identities for the person
- **THEN** the page MUST render labeled external links for each supported provider using the provider's person-specific destination URL format

#### Scenario: External identities are unavailable
- **WHEN** a person detail response does not include any supported external identities
- **THEN** the page MUST omit the external links section without showing disabled actions

### Requirement: Person detail API returns profile and works data in one read
The system SHALL provide a catalog person detail API that returns the profile data needed for the person page together with locally linked works.

#### Scenario: Fetch person detail
- **WHEN** an authenticated client requests a valid catalog person ID from the person detail API
- **THEN** the response MUST include the person's identity, portrait URL when known, biography and birth facts when known, supported external identities, and related local catalog items ordered for browsing

#### Scenario: Person detail not found
- **WHEN** an authenticated client requests a catalog person ID that does not exist
- **THEN** the API MUST return the app's normal not-found behavior so the frontend can render a recoverable missing-page state

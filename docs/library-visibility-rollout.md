# Library Visibility Rollout

`library-tag-role-visibility` introduces library-scoped visibility rules based on reusable access tags and role-level allow/deny rules.

## Default Behavior

- A library with no access tags remains visible to all authenticated users.
- If a user has no `allow` rules across all assigned roles, access remains default-open except where a `deny` rule matches.
- `deny` always overrides `allow`.
- When the same metadata item exists in multiple libraries, the item stays visible if at least one accessible library still contains it.
- Resource lists and playback candidates are still cut down to accessible libraries only.

## Operator Expectations

- You can roll this out gradually.
- Existing users keep their current access until you start assigning access tags or role rules.
- Tagging only a subset of libraries is safe because untagged libraries continue to follow the default-open policy.
- Role visibility is evaluated from the full assigned role set, not only the legacy primary `role` field.

## Recommended Rollout Order

1. Upgrade and deploy the backend with the visibility feature enabled.
2. Leave all libraries untagged at first and verify existing users still see expected content.
3. Create a small set of reusable access tags such as `kids`, `family`, or `adult`.
4. Assign tags to one or two non-critical libraries first.
5. Add role-level `allow` and `deny` rules for a test user group.
6. Verify browse, home, favorites, detail, and playback behavior with those users before widening adoption.

## Direct Request Behavior

- Requests that explicitly target a blocked `library_id` return `403 Forbidden`.
- Non-library-scoped content reads are filtered to the caller's accessible library set.

## Rollback

If you need to disable enforcement without losing configured data:

- Set environment variable `MIBO_DISABLE_LIBRARY_VISIBILITY_ENFORCEMENT=true`
- Redeploy the server

When this flag is enabled:

- Stored library tags remain in the database
- Stored role allow/deny rules remain in the database
- Visibility enforcement is bypassed and authenticated users return to default-open access

After rollback, you can later re-enable enforcement by removing the flag and restarting the service.

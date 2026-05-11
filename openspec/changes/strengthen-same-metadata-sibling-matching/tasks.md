## 1. Matching Model

- [x] 1.1 Add a reusable same-metadata sibling matching model in `mibo-media-server/internal/library` that can represent same-work, same-version, same-episode, supplemental, unresolved, and conflict outcomes with evidence and confidence.
- [x] 1.2 Split canonical identity matching from variant-trait matching so release or edition traits cannot create same-metadata identity by themselves.

## 2. Strong Identity Evidence

- [x] 2.1 Promote sidecar/provider identity, episode tuple, and accepted work-group outputs into primary same-metadata sibling evidence before auto-linking resources.
- [x] 2.2 Consume file `md5` as a strong non-blocking sibling evidence signal when it is available and keep matching functional when it is absent.

## 3. Resource Linking

- [x] 3.1 Route accepted movie scan results through the sibling matcher so same-movie resources reuse one metadata identity and version links instead of creating duplicates.
- [x] 3.2 Route accepted episode scan results through the sibling matcher so same-episode resources reuse one episode metadata identity and version links instead of creating duplicates.
- [x] 3.3 Keep weak or conflicting sibling candidates unresolved or review-required rather than auto-linking them to existing metadata identities.

## 4. Supplemental Isolation And Browse Upgrade

- [x] 4.1 Ensure samples, trailers, featurettes, and other supplemental files do not auto-merge into main primary/version resource sets.
- [x] 4.2 Refresh affected projection and browse upgrade paths after accepted sibling matches so organizing cards collapse into existing metadata-backed cards when appropriate.

## 5. Regression Coverage

- [x] 5.1 Add focused sibling matching tests for same-movie versions, same-episode versions, weak title-only candidates, and provider or sidecar conflicts.
- [x] 5.2 Add `md5`-driven tests for cross-source duplicate recognition and conflict handling when `md5` and metadata evidence disagree.
- [x] 5.3 Add browse or API regression tests proving accepted sibling matches prefer the existing metadata card over a duplicate organizing entry.

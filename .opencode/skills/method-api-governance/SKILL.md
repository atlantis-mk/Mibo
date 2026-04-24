---
name: method-api-governance
description: Review or shape new methods and APIs in MediaWeave using the project's governance rules for necessity, reuse, scope, lifecycle, and deprecation planning.
license: MIT
compatibility: opencode
metadata:
  audience: maintainers
  workflow: architecture-review
---

## What I do

- Evaluate whether a new method or API should be added at all.
- Check for overlap with existing implementations, interfaces, or entry points.
- Turn the repository's governance document into concrete review steps and acceptance criteria.
- Require clear caller, business scenario, ownership, and deprecation planning before approving additions.
- Help converge duplicate capabilities toward one primary entry point.

## When to use me

Use this when a task proposes a new method, helper, service abstraction, facade, or API endpoint.
Use this when reviewing whether an existing method or API is redundant, premature, overlapping, or ready for deprecation.
Use this when a PR needs governance-based justification for adding or keeping an interface.

## Inputs I expect

- The proposed method or API change.
- The business scenario it serves.
- The real caller or consumer, or a concrete integration plan.
- Any existing implementation, endpoint, or documentation that may already cover the same capability.

## Workflow

1. Confirm the business scenario first:
   - Who will call it.
   - What problem it solves.
   - Why existing capability cannot satisfy the need.
2. Search existing code and API documentation before allowing a new addition.
3. For new methods, check all admission conditions:
   - No equivalent implementation already exists.
   - The caller is real and current, not hypothetical.
   - Responsibility boundary is clear.
   - Naming expresses the business action and target accurately.
   - Parameters are not overdesigned for speculative future use.
4. For new APIs, check all admission conditions:
   - Existing APIs cannot reasonably carry the business capability.
   - There is a clear consumer or onboarding plan.
   - Semantics do not overlap with another peer endpoint.
   - Response shape serves the business scenario instead of exposing raw underlying structures.
   - Lifecycle and ownership are explicit.
5. Apply the repository's design constraints:
   - Prefer scenario-driven design over speculative abstraction.
   - Delay abstraction until there are 2-3 real repetition points.
   - Keep one primary entry point for the same capability.
   - Model business actions instead of mirroring storage tables, page buttons, or low-level RPC calls.
6. If an addition is justified, require the proposal or PR to state:
   - Why the addition is necessary.
   - Who the caller or consumer is.
   - How it differs from existing capability.
   - What alternatives were considered.
   - What the deprecation or compatibility plan is.
7. If an existing method or API is obsolete, define the retirement path:
   - Mark it `deprecated`.
   - Document the replacement.
   - Notify dependents to migrate.
   - Set a deletion date.
   - Remove it when the deadline is reached.

## Review outcomes

- Approve when the capability is necessary, non-overlapping, and has a clear boundary.
- Request changes when the capability is needed but naming, scope, reuse, or lifecycle design should be tightened.
- Reject when the proposal has no real consumer, duplicates an existing path, or reflects obvious overdesign.

## Output requirements

- Give a clear governance conclusion: approve, revise, or reject.
- Cite the concrete reason using necessity, duplication, abstraction timing, semantics, or lifecycle.
- When recommending approval, include the required justification items for the PR description.
- When recommending rejection or cleanup, identify the existing capability that should remain the primary entry point.
- Keep recommendations actionable and tied to MediaWeave's governance rules.

## Guardrails

- Do not approve methods or APIs created only for possible future use.
- Do not preserve multiple near-identical entry points such as renamed variants of the same capability.
- Do not move single-use logic into a shared `util` layer without stable reuse value.
- Do not introduce a new API version without a real compatibility break and a migration plan.
- Do not leave deprecated methods or interfaces without replacement guidance and a removal timeline.
- Do not treat different page buttons or caller preferences as sufficient reason to split semantically identical APIs.

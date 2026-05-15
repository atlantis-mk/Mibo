# Agent Orchestration

## Available Agents

| Agent | Purpose | When to Use |
|-------|---------|-------------|
| a11y-architect | Accessibility architecture and audits | UI components, design systems, accessibility reviews |
| architect | System design | Architectural decisions |
| build-error-resolver | Fix build and type errors | When JS or TS builds fail |
| chief-of-staff | Message triage and reply drafting | Email, Slack, LINE, Messenger workflows |
| code-architect | Feature architecture design | Before implementing complex features |
| code-explorer | Deep codebase analysis | Tracing existing behavior and dependencies |
| code-reviewer | General code review | After writing code |
| code-simplifier | Simplify modified code | When code works but is too complex |
| comment-analyzer | Comment quality review | Checking comment accuracy and rot risk |
| conversation-analyzer | Conversation behavior analysis | Deriving hook opportunities from transcripts |
| cpp-build-resolver | C++ build and compile fixes | CMake, linker, template, compile failures |
| cpp-reviewer | C++ code review | C++ projects |
| csharp-reviewer | C# code review | .NET projects |
| database-reviewer | Database review and optimization | SQL, schema, query, and migration work |
| django-build-resolver | Django setup and build fixes | Python dependency, import, migration failures |
| django-reviewer | Django code review | Django and DRF projects |
| doc-updater | Documentation updates | README, codemaps, and guides |
| e2e-runner | E2E testing | Critical user flows |
| explore | Fast repository exploration | Finding files, symbols, and usage quickly |
| fastapi-reviewer | FastAPI code review | FastAPI and Pydantic projects |
| flutter-reviewer | Flutter and Dart code review | Flutter projects |
| fsharp-reviewer | F# code review | F# projects |
| gan-evaluator | Product evaluation loop | Scoring live app behavior against a rubric |
| gan-generator | Product implementation loop | Iterating on features from evaluator feedback |
| gan-planner | Product specification planning | Expanding a short prompt into a build spec |
| general | General multi-step execution | Broad research or implementation support |
| go-build-resolver | Go build and vet fixes | Go compilation or vet failures |
| go-reviewer | Go code review | Go projects |
| harmonyos-app-resolver | HarmonyOS app development review | HarmonyOS or ArkTS projects |
| healthcare-reviewer | Healthcare domain review | Clinical safety or PHI-sensitive systems |
| homelab-architect | Homelab and small-network design | Safe staged home lab changes |
| java-build-resolver | Java build and dependency fixes | Maven, Gradle, compiler failures |
| java-reviewer | Java code review | Spring Boot or Quarkus projects |
| kotlin-build-resolver | Kotlin and Gradle fixes | Kotlin compiler or Gradle failures |
| kotlin-reviewer | Kotlin code review | Kotlin, Android, or KMP projects |
| loop-operator | Autonomous loop supervision | Monitoring and intervening in agent loops |
| mle-reviewer | ML engineering review | Training, inference, MLOps, evaluation changes |
| network-architect | Network architecture design | Enterprise or multi-site network planning |
| network-config-reviewer | Network config review | Router and switch config validation |
| network-troubleshooter | Network issue diagnosis | Connectivity, routing, DNS, policy debugging |
| opensource-forker | Open-source fork preparation | Sanitizing an internal repo for release |
| opensource-packager | Open-source packaging | README, LICENSE, setup, contribution files |
| opensource-sanitizer | Open-source release verification | Secret, PII, and internal reference scanning |
| performance-optimizer | Performance optimization | Bottleneck and runtime analysis |
| planner | Implementation planning | Complex features, refactoring |
| pr-test-analyzer | PR test quality review | Checking whether tests cover real risks |
| python-reviewer | Python code review | Python projects |
| pytorch-build-resolver | PyTorch runtime fixes | Tensor, CUDA, DataLoader, AMP failures |
| refactor-cleaner | Dead code cleanup | Code maintenance |
| rust-build-resolver | Rust build fixes | Cargo, borrow checker, compilation failures |
| rust-reviewer | Rust code review | Rust projects |
| security-reviewer | Security analysis | Auth, input handling, API, sensitive flows |
| seo-specialist | SEO review | Technical SEO and search optimization work |
| silent-failure-hunter | Silent failure analysis | Missing error handling and swallowed failures |
| swift-build-resolver | Swift and Xcode fixes | Swift build, SPM, signing failures |
| swift-reviewer | Swift code review | Swift projects |
| tdd-guide | Test-driven development | New features, bug fixes, refactors |
| type-design-analyzer | Type design review | Checking invariants and type ergonomics |
| typescript-reviewer | TypeScript and JavaScript code review | TS or JS projects |

## Immediate Agent Usage

No user prompt needed:
1. Complex feature requests - Use **planner** agent
2. Code just written/modified - Use **code-reviewer** agent
3. Bug fix or new feature - Use **tdd-guide** agent
4. Architectural decision - Use **architect** agent
5. Build/type failure - Use the matching `*-build-resolver` agent
6. Language-specific code changes - Use the matching `*-reviewer` agent
7. Security-sensitive changes - Use **security-reviewer** agent
8. UI or design-system work - Use **a11y-architect** agent

## Parallel Task Execution

ALWAYS use parallel Task execution for independent operations:

```markdown
# GOOD: Parallel execution
Launch 3 agents in parallel:
1. Agent 1: Security analysis of auth module
2. Agent 2: Performance review of cache system
3. Agent 3: Type checking of utilities

# BAD: Sequential when unnecessary
First agent 1, then agent 2, then agent 3
```

## Multi-Perspective Analysis

For complex problems, use split role sub-agents:
- Factual reviewer
- Senior engineer
- Security expert
- Consistency reviewer
- Redundancy checker

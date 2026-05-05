## ADDED Requirements

### Requirement: Workflow Runs Represent Library Work
The system SHALL represent each library ingest, refresh, scheduled scan, and storage-change refresh as a durable workflow run scoped to a library, reason, priority, and lifecycle status.

#### Scenario: Manual library scan creates a run
- **WHEN** a user requests a scan for a library
- **THEN** the system MUST create or reuse an active workflow run for that library and scan reason
- **AND** the run MUST expose queued, running, completed, failed, cancelled, or superseded status

#### Scenario: Different libraries have independent runs
- **WHEN** library A and library B are scanned at the same time
- **THEN** the system MUST represent them as separate workflow runs
- **AND** progress or failure in one run MUST NOT block status updates for the other run

### Requirement: Workflow Tasks Form A Dependency DAG
The system SHALL decompose workflow runs into durable tasks with explicit dependencies so a task becomes claimable only after all required predecessor tasks have completed successfully.

#### Scenario: Scan discovery unlocks materialization
- **WHEN** a discovery task completes for a workflow run
- **THEN** dependent materialization tasks MUST become eligible for scheduling if their other dependencies are complete

#### Scenario: Dependency failure blocks descendants
- **WHEN** a required predecessor task fails and is not retryable
- **THEN** dependent tasks MUST remain blocked or be marked skipped
- **AND** the workflow run MUST expose the failure cause

### Requirement: Scheduler Selects Tasks By Readiness Fairness And FIFO
The scheduler SHALL claim only queued tasks whose dependencies are satisfied, whose available time has arrived, whose priority is eligible, and whose ordering respects priority, per-run fairness, and FIFO order within equal priority and fairness class.

#### Scenario: Older equal-priority task wins within a pool
- **WHEN** two ready tasks have the same priority, compatible resources, and no fairness difference
- **THEN** the scheduler MUST claim the task that entered the ready queue first

#### Scenario: Large run does not starve small run
- **WHEN** one workflow run has many ready batch tasks and another run has a ready task
- **THEN** the scheduler MUST rotate work fairly enough that the second run can claim capacity instead of waiting for all tasks from the first run to finish

### Requirement: Scheduler Enforces Resource Budgets
The scheduler SHALL require each task type to declare resource requirements and MUST NOT run a task when claiming it would exceed any configured resource budget.

#### Scenario: ffprobe capacity is full
- **WHEN** all configured `ffprobe` capacity is consumed by running probe tasks
- **THEN** the scheduler MUST NOT claim another task requiring `ffprobe`
- **AND** it MAY continue claiming tasks that do not require `ffprobe` when their own resources are available

#### Scenario: database write capacity is constrained
- **WHEN** the configured `db_write` budget is exhausted
- **THEN** the scheduler MUST defer additional mutating tasks until capacity is released

### Requirement: Scheduler Enforces Library Safety
The scheduler SHALL prevent incompatible mutating tasks for the same library from running concurrently while allowing compatible tasks for different libraries to run concurrently when resources permit.

#### Scenario: Same library has two mutating tasks
- **WHEN** a mutating task for library A is running
- **THEN** the scheduler MUST NOT claim another incompatible mutating task for library A

#### Scenario: Different libraries can run concurrently
- **WHEN** a mutating task for library A is running and a ready mutating task for library B has available resources
- **THEN** the scheduler MUST be able to claim the library B task without waiting for library A to finish

### Requirement: Tasks Use Leases For Recovery
The system SHALL claim workflow tasks with leases, renew leases for long-running tasks, and make expired leases recoverable so tasks are not stranded after worker crashes.

#### Scenario: Worker crashes while task is leased
- **WHEN** a task is running and its worker stops renewing the lease
- **THEN** the task MUST become eligible for retry or recovery after `lease_until` expires

#### Scenario: Long task keeps lease alive
- **WHEN** a long-running task is still healthy
- **THEN** the worker MUST renew the task lease before expiration

### Requirement: Workflow Cancellation Stops Future Work
The system SHALL support cancelling a workflow run, request cancellation of running tasks, and prevent unstarted tasks in the run from being claimed.

#### Scenario: Administrator cancels a run
- **WHEN** an administrator cancels a running workflow run
- **THEN** queued tasks in that run MUST be cancelled or skipped
- **AND** running tasks MUST observe cancellation and stop at the next safe cancellation point

### Requirement: Workflow Progress Is Observable
The system SHALL expose enough workflow run and task status for administrators and clients to understand queued, running, blocked, completed, failed, cancelled, and resource-waiting work.

#### Scenario: User views scan progress
- **WHEN** a library scan workflow is active
- **THEN** the system MUST expose the current stages, counts of tasks by status, recent error summary, and active resource waits for that run

### Requirement: Existing Job APIs Remain Compatible During Migration
The system SHALL preserve existing scan trigger behavior and job status compatibility while workflow-backed execution is introduced.

#### Scenario: Existing scan endpoint queues workflow work
- **WHEN** a client calls an existing library scan endpoint
- **THEN** the endpoint MUST continue returning an accepted response
- **AND** the queued backend work MUST be trackable through either existing job status compatibility or workflow status visibility

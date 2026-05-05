## ADDED Requirements

### Requirement: Video Probing Participates In Workflow Scheduling
The system SHALL execute background inventory video probing as workflow tasks that declare ffprobe, disk-read, CPU, and database-write resources while preserving existing technical metadata capture semantics.

#### Scenario: ffprobe slots are available
- **WHEN** probe workflow tasks are ready and ffprobe resource capacity is available
- **THEN** the scheduler MUST run probe tasks up to the configured ffprobe budget

#### Scenario: ffprobe slots are exhausted
- **WHEN** all ffprobe capacity is occupied
- **THEN** additional probe workflow tasks MUST wait without blocking scan discovery or other tasks that do not require ffprobe capacity

## MODIFIED Requirements

### Requirement: Detailed video stream attributes are captured
The system SHALL capture detailed technical attributes for catalog video streams from probe data when those attributes are available, regardless of whether the probe was initiated by a legacy job or a workflow task.

#### Scenario: Probe returns detailed video stream metadata
- **WHEN** an inventory file probe returns video stream fields including codec, profile, level, dimensions, frame rate, field order, stream bitrate, color space, bit depth or pixel format, and reference frames
- **THEN** the system MUST persist those values on the corresponding media stream record without losing existing stream identity, language, title, duration, or dimensions

#### Scenario: Probe omits optional technical fields
- **WHEN** an inventory file probe returns a video stream without one or more detailed technical attributes
- **THEN** the system MUST persist the available values and leave missing detailed attributes empty without failing the probe task

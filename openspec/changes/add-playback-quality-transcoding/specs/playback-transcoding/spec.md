## ADDED Requirements

### Requirement: Transcoded playback uses browser-compatible HLS
The backend SHALL provide transcoded playback variants as HLS streams whose video and audio codecs are compatible with the browser client profile.

#### Scenario: Audio-only repair stream
- **WHEN** the original video codec is browser-compatible but one or more selected audio tracks are not browser-compatible
- **THEN** the system SHALL create an HLS variant that copies the video stream and transcodes audio to AAC

#### Scenario: Full quality transcode stream
- **WHEN** a selected quality variant requires video conversion or scaling
- **THEN** the system SHALL create an HLS variant using browser-compatible H.264 video and AAC audio

#### Scenario: Generated playback starts before full completion
- **WHEN** a transcode session has produced the playlist and initial playable segments
- **THEN** the frontend SHALL be able to start playback before the entire media item has finished transcoding

### Requirement: Transcode sessions are authorized and provider-safe
The backend SHALL start and serve transcode sessions only after the requesting user passes playback authorization and library visibility checks.

#### Scenario: Authorized user requests a transcode variant
- **WHEN** an authorized user requests a transcode variant for an accessible media file
- **THEN** the backend SHALL resolve source access through the existing access layer and return a session-scoped HLS manifest URL

#### Scenario: Unauthorized segment request
- **WHEN** a user requests a transcode manifest or segment without a valid session grant
- **THEN** the backend SHALL reject the request without exposing local paths, provider URLs, or generated media content

#### Scenario: Cloud-drive source access expires
- **WHEN** a remote provider access URL expires before or during transcode startup
- **THEN** the backend SHALL refresh source access through the provider/access layer or fail with a retryable playback error

### Requirement: Transcode sessions support seek and cleanup
The backend SHALL manage transcoding as short-lived sessions that support seeking to uncached positions and clean up generated output after inactivity.

#### Scenario: Seek to cached segment
- **WHEN** the user seeks to a position whose segments are already generated for the active session
- **THEN** the player SHALL continue from cached HLS output without starting a new FFmpeg process

#### Scenario: Seek to uncached segment
- **WHEN** the user seeks to a position whose segments are not available
- **THEN** the backend SHALL restart or retarget the transcode session near the requested timestamp and expose a refreshed HLS playlist once initial segments are ready

#### Scenario: Session becomes idle
- **WHEN** a transcode session has no manifest or segment access for longer than the configured idle timeout
- **THEN** the backend SHALL stop the FFmpeg process and remove generated temporary files for that session

### Requirement: Encoder planning supports CPU and optional GPU acceleration
The backend SHALL choose FFmpeg arguments from a safe server-side encoder plan rather than accepting raw encoder commands from the client.

#### Scenario: Audio repair does not re-encode video
- **WHEN** the selected variant is audio repair
- **THEN** the FFmpeg plan SHALL use video stream copy and AAC audio encoding

#### Scenario: Hardware encoder is available
- **WHEN** a full video transcode is requested and a configured hardware H.264 encoder is available
- **THEN** the FFmpeg plan SHALL use that hardware encoder with browser-compatible HLS output settings

#### Scenario: Hardware encoder is unavailable
- **WHEN** a full video transcode is requested and no configured hardware encoder is available
- **THEN** the FFmpeg plan SHALL fall back to CPU H.264 encoding with safe default quality and bitrate settings

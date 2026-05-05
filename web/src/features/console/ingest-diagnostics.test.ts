import { describe, expect, it } from "vitest"

import type { ConsoleSummary, IngestDiagnosticsResult } from "#/lib/mibo-api"

function summary(): ConsoleSummary {
  return {
    server: {
      name: "Mibo",
      service: "mibo-media-server",
      status: "ok",
      version: "unknown",
      update_status: "unknown",
      api_address: ":8080",
      port: 8080,
      uptime_seconds: 1,
      storage_provider: "local",
      storage_root: "/media",
      database_driver: "sqlite",
    },
    access: { addresses: [] },
    media: {
      libraries: 1,
      media_sources: 1,
      catalog_items: 1,
      inventory_files: 1,
      movies: 1,
      series: 0,
      episodes: 0,
      people: 0,
      active_jobs: 0,
      failed_jobs: 0,
      schedules: 0,
      enabled_schedules: 0,
      warnings: 0,
      ingest: {
        organizing: 2,
        failed: 1,
        stale: 1,
        review_required: 1,
        retry_eligible: 2,
      },
    },
    health: {
      database: { status: "ok" },
      storage: { status: "ok" },
      modules: [],
    },
    devices: [],
    quick_actions: [],
    activity: [],
    warnings: [],
  }
}

describe("console ingest diagnostics contracts", () => {
  it("carries ingest health counts on the console summary", () => {
    const data = summary()
    expect(data.media.ingest?.failed).toBe(1)
    expect(data.media.ingest?.retry_eligible).toBe(2)
  })

  it("represents retryable diagnostic stages", () => {
    const diagnostics: IngestDiagnosticsResult = {
      summary: summary().media.ingest!,
      stages: [
        {
          id: 7,
          unit_key: "inventory_file:9",
          library_id: 1,
          inventory_file_id: 9,
          storage_path: "/media/Movie.mkv",
          condition_type: "probed",
          status: "failed",
          reason: "probe_failed",
          message: "probe failed",
          severity: "error",
          attempts: 1,
          retry_eligible: true,
          stale: false,
          updated_at: new Date().toISOString(),
        },
      ],
    }

    expect(diagnostics.stages[0].retry_eligible).toBe(true)
    expect(diagnostics.stages[0].storage_path).toContain("Movie")
  })
})

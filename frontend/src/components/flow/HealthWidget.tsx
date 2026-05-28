"use client"

import { useEffect, useState } from "react"
import { api } from "@/lib/api/client"
import { HealthReport, ServiceHealth } from "@/types"

function StatusDot({
  status,
}: {
  status: string
}) {
  const colors = {
    healthy: "bg-green-500",
    degraded: "bg-yellow-500",
    down: "bg-red-500",
    unknown: "bg-gray-500",
  }

  return (
    <div
      className={`w-2 h-2 rounded-full ${
        colors[status as keyof typeof colors] ||
        colors.unknown
      }`}
    />
  )
}

export function HealthWidget() {
  const [report, setReport] =
    useState<HealthReport | null>(null)
  const [loading, setLoading] = useState(true)

  async function fetchHealth() {
    try {
      const { data } = await api.get(
        "/flow/health"
      )
      setReport(data.data)
    } catch {
      // Backend may not be ready
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchHealth()
    const interval = setInterval(
      fetchHealth, 60000
    )
    return () => clearInterval(interval)
  }, [])

  if (loading) {
    return (
      <div className="space-y-2">
        <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          Health
        </p>
        {[1, 2, 3].map((i) => (
          <div
            key={i}
            className="h-8 bg-secondary rounded-md animate-pulse"
          />
        ))}
      </div>
    )
  }

  if (!report) {
    return (
      <div className="space-y-2">
        <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          Health
        </p>
        <div className="text-xs text-muted-foreground p-3 bg-secondary rounded-md">
          Unable to connect to backend
        </div>
      </div>
    )
  }

  const overallColor = {
    healthy: "text-green-400",
    degraded: "text-yellow-400",
    down: "text-red-400",
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          Health
        </p>
        <span
          className={`text-xs font-medium ${
            overallColor[
              report.overall as keyof typeof overallColor
            ] || "text-muted-foreground"
          }`}
        >
          {report.overall}
        </span>
      </div>

      <div className="space-y-1">
        {Object.entries(report.services).map(
          ([key, svc]) => (
            <div
              key={key}
              className="flex items-center justify-between py-2 px-3 rounded-md bg-secondary"
            >
              <div className="flex items-center gap-2">
                <StatusDot status={svc.status} />
                <span className="text-xs">
                  {svc.name || key}
                </span>
              </div>
              <div className="text-right">
                {svc.quota_max ? (
                  <div>
                    <div className="text-xs text-muted-foreground">
                      {svc.quota_used}/
                      {svc.quota_max}
                    </div>
                    <div className="w-16 h-1 bg-border rounded-full mt-1">
                      <div
                        className={`h-1 rounded-full ${
                          (svc.quota_pct || 0) > 90
                            ? "bg-red-500"
                            : (svc.quota_pct || 0) > 75
                            ? "bg-yellow-500"
                            : "bg-green-500"
                        }`}
                        style={{
                          width: `${Math.min(
                            svc.quota_pct || 0,
                            100
                          )}%`,
                        }}
                      />
                    </div>
                  </div>
                ) : (
                  <span className="text-xs text-muted-foreground">
                    {svc.latency_ms}ms
                  </span>
                )}
              </div>
            </div>
          )
        )}
      </div>
    </div>
  )
}

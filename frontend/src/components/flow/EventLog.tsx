"use client"

import { useEffect, useState } from "react"
import { api } from "@/lib/api/client"
import { FlowEvent } from "@/types"
import { formatRelative } from "@/lib/utils"

const eventColors: Record<string, string> = {
  "form.submitted": "text-purple-400",
  "form.published": "text-purple-300",
  "mail.contact_added": "text-green-400",
  "mail.campaign_sent": "text-green-300",
  "link.clicked": "text-cyan-400",
  "link.created": "text-cyan-300",
  "doc.viewed": "text-yellow-400",
  "doc.published": "text-yellow-300",
  "crm.contact_created": "text-blue-400",
  "system.health_check": "text-gray-400",
}

export function EventLog() {
  const [events, setEvents] = useState<
    FlowEvent[]
  >([])
  const [loading, setLoading] = useState(true)

  async function fetchEvents() {
    try {
      const { data } = await api.get(
        "/flow/events?limit=20"
      )
      setEvents(data.data || [])
    } catch {
      // silently fail
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchEvents()
    const interval = setInterval(
      fetchEvents, 5000
    )
    return () => clearInterval(interval)
  }, [])

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
          Event Log
        </p>
        <button
          onClick={fetchEvents}
          className="text-xs text-muted-foreground hover:text-foreground transition-colors"
        >
          ↻
        </button>
      </div>

      {loading ? (
        <div className="space-y-1">
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-10 bg-secondary rounded animate-pulse"
            />
          ))}
        </div>
      ) : events.length === 0 ? (
        <div className="text-xs text-muted-foreground p-3 bg-secondary rounded-md text-center">
          No events yet.
          <br />
          Submit a form to see events.
        </div>
      ) : (
        <div className="space-y-1 max-h-64 overflow-y-auto">
          {events.map((event) => (
            <div
              key={event.id}
              className="px-3 py-2 rounded-md bg-secondary hover:bg-secondary/80 transition-colors"
            >
              <div className="flex items-center justify-between gap-2">
                <span
                  className={`text-xs font-mono truncate ${
                    eventColors[event.type] ||
                    "text-muted-foreground"
                  }`}
                >
                  {event.type}
                </span>
                <span className="text-xs text-muted-foreground flex-shrink-0">
                  {formatRelative(event.occurred_at)}
                </span>
              </div>
              <span className="text-xs text-muted-foreground">
                from {event.source}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

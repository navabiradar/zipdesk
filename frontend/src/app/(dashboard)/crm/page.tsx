"use client"

import { useState, useEffect } from "react"
import { api } from "@/lib/api/client"
import { CRMContact } from "@/types"
import { formatRelative } from "@/lib/utils"

export default function CRMPage() {
  const [contacts, setContacts] = useState<
    CRMContact[]
  >([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    async function fetch() {
      try {
        const { data } = await api.get(
          "/crm/contacts?page=1&per_page=20"
        )
        setContacts(data.data || [])
        setTotal(data.meta?.total || 0)
      } catch {
        //
      } finally {
        setLoading(false)
      }
    }
    fetch()
  }, [])

  const statusColors: Record<string, string> = {
    new: "text-blue-400",
    contacted: "text-yellow-400",
    qualified: "text-green-400",
    lost: "text-red-400",
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="mb-6">
        <h2 className="text-lg font-semibold">CRM</h2>
        <p className="text-sm text-muted-foreground">
          {total} contacts
        </p>
      </div>

      {loading ? (
        <div className="space-y-2">
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-16 bg-secondary rounded-xl animate-pulse"
            />
          ))}
        </div>
      ) : contacts.length === 0 ? (
        <div className="text-center py-16 text-muted-foreground">
          <p className="text-4xl mb-3">👥</p>
          <p className="text-sm">
            No CRM contacts yet.
            <br />
            Submit a form to auto-create contacts.
          </p>
        </div>
      ) : (
        <div className="space-y-1">
          {contacts.map((c) => (
            <div
              key={c.id}
              className="flex items-center gap-4 px-4 py-3 bg-card border border-border rounded-lg"
            >
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium">
                  {c.first_name} {c.last_name}
                </p>
                <p className="text-xs text-muted-foreground">
                  {c.email}
                </p>
              </div>
              <div className="flex items-center gap-3 text-xs">
                <span
                  className={
                    statusColors[c.lead_status] ||
                    "text-muted-foreground"
                  }
                >
                  {c.lead_status}
                </span>
                <span className="text-muted-foreground">
                  Score: {c.lead_score}
                </span>
                <span className="text-muted-foreground">
                  {formatRelative(c.created_at)}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

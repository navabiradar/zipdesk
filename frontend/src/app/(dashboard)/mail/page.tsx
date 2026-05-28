"use client"

import { useState, useEffect } from "react"
import { api } from "@/lib/api/client"
import { MailContact } from "@/types"
import { Input } from "@/components/ui/input"
import { formatRelative } from "@/lib/utils"

export default function MailPage() {
  const [contacts, setContacts] = useState<
    MailContact[]
  >([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState("")

  async function fetchContacts(q = "") {
    try {
      const { data } = await api.get(
        `/mail/contacts?page=1&per_page=20&search=${q}`
      )
      setContacts(data.data || [])
      setTotal(data.meta?.total || 0)
    } catch {
      //
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchContacts()
  }, [])

  const sourceColors: Record<string, string> = {
    form: "bg-purple-500/15 text-purple-400 border-purple-500/25",
    manual: "bg-secondary text-muted-foreground",
    api: "bg-blue-500/15 text-blue-400 border-blue-500/25",
    import: "bg-yellow-500/15 text-yellow-400 border-yellow-500/25",
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-lg font-semibold">
            Mail Contacts
          </h2>
          <p className="text-sm text-muted-foreground">
            {total} total contacts
          </p>
        </div>
      </div>

      {/* Search */}
      <div className="mb-4">
        <Input
          placeholder="Search contacts..."
          value={search}
          onChange={(e) => {
            setSearch(e.target.value)
            fetchContacts(e.target.value)
          }}
          className="max-w-xs"
        />
      </div>

      {loading ? (
        <div className="space-y-2">
          {[1, 2, 3, 4, 5].map((i) => (
            <div
              key={i}
              className="h-14 bg-secondary rounded-xl animate-pulse"
            />
          ))}
        </div>
      ) : contacts.length === 0 ? (
        <div className="text-center py-16 text-muted-foreground">
          <p className="text-4xl mb-3">📧</p>
          <p className="text-sm">
            No contacts yet.
            <br />
            Submit a form to auto-create contacts.
          </p>
        </div>
      ) : (
        <div className="space-y-1">
          {contacts.map((contact) => (
            <div
              key={contact.id}
              className="flex items-center gap-4 px-4 py-3 bg-card border border-border rounded-lg hover:bg-secondary/30 transition-colors"
            >
              <div className="w-8 h-8 rounded-full bg-secondary flex items-center justify-center flex-shrink-0">
                <span className="text-xs font-medium">
                  {(contact.first_name ||
                    contact.email)?.[0]
                    ?.toUpperCase() || "?"}
                </span>
              </div>

              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate">
                  {contact.first_name && contact.last_name
                    ? `${contact.first_name} ${contact.last_name}`
                    : contact.email}
                </p>
                {(contact.first_name ||
                  contact.last_name) && (
                  <p className="text-xs text-muted-foreground">
                    {contact.email}
                  </p>
                )}
              </div>

              <div className="flex items-center gap-2 flex-shrink-0">
                <span
                  className={`text-xs px-2 py-0.5 rounded-full border ${
                    sourceColors[contact.source] ||
                    sourceColors.manual
                  }`}
                >
                  {contact.source}
                </span>
                <span className="text-xs text-muted-foreground">
                  {formatRelative(contact.created_at)}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

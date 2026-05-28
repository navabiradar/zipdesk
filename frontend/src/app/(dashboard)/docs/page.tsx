"use client"

import { useState, useEffect } from "react"
import { api } from "@/lib/api/client"
import { Document } from "@/types"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { formatRelative } from "@/lib/utils"

export default function DocsPage() {
  const [docs, setDocs] = useState<Document[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [title, setTitle] = useState("")
  const [creating, setCreating] = useState(false)

  async function fetchDocs() {
    try {
      const { data } = await api.get("/docs")
      setDocs(data.data || [])
    } catch {
      //
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchDocs()
  }, [])

  async function createDoc(e: React.FormEvent) {
    e.preventDefault()
    setCreating(true)
    try {
      await api.post("/docs", { title })
      setTitle("")
      setShowCreate(false)
      fetchDocs()
    } catch {
      alert("Failed to create document")
    } finally {
      setCreating(false)
    }
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-lg font-semibold">
            Docs
          </h2>
          <p className="text-sm text-muted-foreground">
            {docs.length} documents
          </p>
        </div>
        <Button
          size="sm"
          onClick={() => setShowCreate(!showCreate)}
        >
          + Create Doc
        </Button>
      </div>

      {showCreate && (
        <form
          onSubmit={createDoc}
          className="mb-6 p-4 bg-card border border-border rounded-xl space-y-3"
        >
          <Input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Document title..."
            required
            autoFocus
          />
          <div className="flex gap-2">
            <Button
              type="submit"
              size="sm"
              disabled={creating}
            >
              Create
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setShowCreate(false)}
            >
              Cancel
            </Button>
          </div>
        </form>
      )}

      {loading ? (
        <div className="space-y-2">
          {[1, 2].map((i) => (
            <div
              key={i}
              className="h-16 bg-secondary rounded-xl animate-pulse"
            />
          ))}
        </div>
      ) : docs.length === 0 ? (
        <div className="text-center py-16 text-muted-foreground">
          <p className="text-4xl mb-3">📄</p>
          <p className="text-sm">
            No documents yet.
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {docs.map((doc) => (
            <div
              key={doc.id}
              className="flex items-center gap-4 p-4 bg-card border border-border rounded-xl"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <p className="text-sm font-medium">
                    {doc.title}
                  </p>
                  <Badge
                    variant={
                      doc.is_published
                        ? "default"
                        : "secondary"
                    }
                    className="text-xs"
                  >
                    {doc.status}
                  </Badge>
                </div>
                <p className="text-xs text-muted-foreground">
                  {formatRelative(doc.created_at)}
                </p>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

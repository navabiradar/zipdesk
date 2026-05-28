"use client"

import { useState, useEffect } from "react"
import { api } from "@/lib/api/client"
import { Link as LinkType } from "@/types"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { copyToClipboard, formatRelative } from "@/lib/utils"

const API_URL =
  process.env.NEXT_PUBLIC_API_URL ||
  "http://localhost:8080"

export default function LinksPage() {
  const [links, setLinks] = useState<LinkType[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [url, setUrl] = useState("")
  const [title, setTitle] = useState("")
  const [creating, setCreating] = useState(false)
  const [copied, setCopied] = useState<string | null>(null)

  async function fetchLinks() {
    try {
      const { data } = await api.get(
        "/links?page=1&per_page=20"
      )
      setLinks(data.data || [])
      setTotal(data.meta?.total || 0)
    } catch {
      //
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchLinks()
  }, [])

  async function createLink(e: React.FormEvent) {
    e.preventDefault()
    setCreating(true)
    try {
      await api.post("/links", {
        original_url: url,
        title,
      })
      setUrl("")
      setTitle("")
      setShowCreate(false)
      fetchLinks()
    } catch (err: any) {
      alert(
        err.response?.data?.error?.message ||
        "Failed to create link"
      )
    } finally {
      setCreating(false)
    }
  }

  async function deleteLink(id: string) {
    if (!confirm("Delete this link?")) return
    await api.delete(`/links/${id}`)
    fetchLinks()
  }

  function copy(shortCode: string) {
    const shortUrl = `${API_URL}/s/${shortCode}`
    copyToClipboard(shortUrl)
    setCopied(shortCode)
    setTimeout(() => setCopied(null), 2000)
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">

      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-lg font-semibold">
            Links
          </h2>
          <p className="text-sm text-muted-foreground">
            {total} total links
          </p>
        </div>
        <Button
          onClick={() => setShowCreate(!showCreate)}
          size="sm"
        >
          + Create Link
        </Button>
      </div>

      {/* Create form */}
      {showCreate && (
        <form
          onSubmit={createLink}
          className="mb-6 p-4 bg-card border border-border rounded-xl space-y-3"
        >
          <h3 className="text-sm font-medium">
            New Short Link
          </h3>
          <div>
            <Label className="text-xs">
              Destination URL
            </Label>
            <Input
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://example.com/long-url"
              required
              autoFocus
            />
          </div>
          <div>
            <Label className="text-xs">
              Title (optional)
            </Label>
            <Input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="My Link"
            />
          </div>
          <div className="flex gap-2">
            <Button
              type="submit"
              size="sm"
              disabled={creating}
            >
              {creating ? "Creating..." : "Create"}
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

      {/* Links table */}
      {loading ? (
        <div className="space-y-2">
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-16 bg-secondary rounded-xl animate-pulse"
            />
          ))}
        </div>
      ) : links.length === 0 ? (
        <div className="text-center py-16 text-muted-foreground">
          <p className="text-4xl mb-3">🔗</p>
          <p className="text-sm">
            No links yet. Create your first short link.
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {links.map((link) => (
            <div
              key={link.id}
              className="flex items-center gap-4 p-4 bg-card border border-border rounded-xl hover:border-border/80 transition-colors"
            >
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate">
                  {link.title || link.short_code}
                </p>
                <p className="text-xs text-muted-foreground truncate">
                  {link.original_url}
                </p>
              </div>

              <div className="flex items-center gap-4 text-xs text-muted-foreground flex-shrink-0">
                <span className="font-mono bg-secondary px-2 py-1 rounded">
                  /s/{link.short_code}
                </span>
                <span>
                  {link.total_clicks} clicks
                </span>
                <span>
                  {formatRelative(link.created_at)}
                </span>
              </div>

              <div className="flex gap-1 flex-shrink-0">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => copy(link.short_code)}
                  className="text-xs"
                >
                  {copied === link.short_code
                    ? "✓"
                    : "Copy"}
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => deleteLink(link.id)}
                  className="text-xs text-red-400 hover:text-red-300"
                >
                  Del
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

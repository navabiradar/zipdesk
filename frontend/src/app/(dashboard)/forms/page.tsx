"use client"

import { useState, useEffect } from "react"
import Link from "next/link"
import { api } from "@/lib/api/client"
import { Form } from "@/types"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { formatRelative } from "@/lib/utils"

export default function FormsPage() {
  const [forms, setForms] = useState<Form[]>([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [title, setTitle] = useState("")
  const [showCreate, setShowCreate] = useState(false)

  async function fetchForms() {
    try {
      const { data } = await api.get("/forms")
      setForms(data.data || [])
    } catch {
      //
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchForms()
  }, [])

  async function createForm(e: React.FormEvent) {
    e.preventDefault()
    setCreating(true)
    try {
      const { data } = await api.post("/forms", {
        title,
        description: "",
      })
      setTitle("")
      setShowCreate(false)
      fetchForms()
    } catch (err: any) {
      alert(
        err.response?.data?.error?.message ||
        "Failed to create form"
      )
    } finally {
      setCreating(false)
    }
  }

  async function deleteForm(id: string) {
    if (!confirm("Delete this form?")) return
    await api.delete(`/forms/${id}`)
    fetchForms()
  }

  async function publishForm(id: string) {
    await api.post(`/forms/${id}/publish`)
    fetchForms()
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-lg font-semibold">
            Forms
          </h2>
          <p className="text-sm text-muted-foreground">
            {forms.length} total forms
          </p>
        </div>
        <Button
          onClick={() => setShowCreate(!showCreate)}
          size="sm"
        >
          + Create Form
        </Button>
      </div>

      {showCreate && (
        <form
          onSubmit={createForm}
          className="mb-6 p-4 bg-card border border-border rounded-xl space-y-3"
        >
          <h3 className="text-sm font-medium">
            New Form
          </h3>
          <Input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Form title..."
            required
            autoFocus
          />
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

      {loading ? (
        <div className="space-y-2">
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-20 bg-secondary rounded-xl animate-pulse"
            />
          ))}
        </div>
      ) : forms.length === 0 ? (
        <div className="text-center py-16 text-muted-foreground">
          <p className="text-4xl mb-3">📝</p>
          <p className="text-sm">
            No forms yet. Create your first form.
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {forms.map((form) => (
            <div
              key={form.id}
              className="flex items-center gap-4 p-4 bg-card border border-border rounded-xl"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <p className="text-sm font-medium">
                    {form.title}
                  </p>
                  <Badge
                    variant={
                      form.is_published
                        ? "default"
                        : "secondary"
                    }
                    className="text-xs"
                  >
                    {form.is_published
                      ? "Live"
                      : "Draft"}
                  </Badge>
                </div>
                <p className="text-xs text-muted-foreground mt-0.5">
                  /f/{form.slug} ·{" "}
                  {formatRelative(form.created_at)}
                </p>
              </div>

              <div className="flex gap-1">
                {!form.is_published ? (
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-xs text-green-400 hover:text-green-300"
                    onClick={() => publishForm(form.id)}
                  >
                    Publish
                  </Button>
                ) : (
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-xs"
                    onClick={() =>
                      window.open(
                        `/f/${form.slug}`,
                        "_blank"
                      )
                    }
                  >
                    View ↗
                  </Button>
                )}
                <Link
                  href={`/forms/${form.id}/responses`}
                >
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-xs"
                  >
                    Responses
                  </Button>
                </Link>
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-xs text-red-400 hover:text-red-300"
                  onClick={() => deleteForm(form.id)}
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

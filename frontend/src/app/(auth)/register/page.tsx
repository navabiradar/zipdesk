"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import Link from "next/link"
import { register } from "@/lib/auth"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

export default function RegisterPage() {
  const router = useRouter()
  const [form, setForm] = useState({
    name: "",
    email: "",
    password: "",
    workspace_name: "",
  })
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  function update(
    field: string,
    value: string
  ) {
    setForm((prev) => ({ ...prev, [field]: value }))
  }

  async function handleSubmit(
    e: React.FormEvent
  ) {
    e.preventDefault()
    setError("")
    setLoading(true)

    try {
      await register(
        form.name,
        form.email,
        form.password,
        form.workspace_name
      )
      router.push("/flow")
    } catch (err: any) {
      setError(
        err.response?.data?.error?.message ||
        "Registration failed"
      )
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="w-full max-w-sm space-y-8 p-8">

        <div className="text-center">
          <div className="inline-flex items-center gap-2 mb-6">
            <div className="w-8 h-8 bg-blue-600 rounded-lg flex items-center justify-center">
              <span className="text-white font-bold text-sm">Z</span>
            </div>
            <span className="text-xl font-semibold">ZipDesk</span>
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">
            Create your workspace
          </h1>
          <p className="text-sm text-muted-foreground mt-1">
            Start your 14-day free trial
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div className="p-3 text-sm text-red-500 bg-red-500/10 border border-red-500/20 rounded-lg">
              {error}
            </div>
          )}

          <div className="space-y-2">
            <Label>Full name</Label>
            <Input
              placeholder="Alex Smith"
              value={form.name}
              onChange={(e) =>
                update("name", e.target.value)
              }
              required
              autoFocus
            />
          </div>

          <div className="space-y-2">
            <Label>Work email</Label>
            <Input
              type="email"
              placeholder="alex@company.com"
              value={form.email}
              onChange={(e) =>
                update("email", e.target.value)
              }
              required
            />
          </div>

          <div className="space-y-2">
            <Label>Workspace name</Label>
            <Input
              placeholder="Acme Inc"
              value={form.workspace_name}
              onChange={(e) =>
                update("workspace_name", e.target.value)
              }
              required
            />
          </div>

          <div className="space-y-2">
            <Label>Password</Label>
            <Input
              type="password"
              placeholder="Min. 8 characters"
              value={form.password}
              onChange={(e) =>
                update("password", e.target.value)
              }
              required
              minLength={8}
            />
          </div>

          <Button
            type="submit"
            className="w-full"
            disabled={loading}
          >
            {loading
              ? "Creating workspace..."
              : "Create workspace →"
            }
          </Button>
        </form>

        <p className="text-center text-sm text-muted-foreground">
          Already have an account?{" "}
          <Link
            href="/login"
            className="text-primary hover:underline"
          >
            Sign in
          </Link>
        </p>
      </div>
    </div>
  )
}

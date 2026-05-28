"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import { cn } from "@/lib/utils"
import { logout } from "@/lib/auth"
import { useAuthStore } from "@/stores/authStore"

const navItems = [
  {
    label: "Flow",
    href: "/flow",
    icon: "⚡",
    highlight: true,
  },
  { label: "Links", href: "/links", icon: "🔗" },
  { label: "Forms", href: "/forms", icon: "📝" },
  { label: "Docs", href: "/docs", icon: "📄" },
  { label: "Mail", href: "/mail", icon: "📧" },
  { label: "CRM", href: "/crm", icon: "👥" },
]

export function Sidebar() {
  const pathname = usePathname()
  const { workspace, user } = useAuthStore()

  return (
    <aside className="w-56 min-h-screen bg-card border-r border-border flex flex-col">

      {/* Logo */}
      <div className="p-4 border-b border-border">
        <div className="flex items-center gap-2">
          <div className="w-7 h-7 bg-blue-600 rounded-md flex items-center justify-center">
            <span className="text-white font-bold text-xs">Z</span>
          </div>
          <div>
            <p className="text-sm font-semibold leading-none">
              ZipDesk
            </p>
            <p className="text-xs text-muted-foreground mt-0.5 truncate max-w-[110px]">
              {workspace?.name || "Workspace"}
            </p>
          </div>
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 p-3 space-y-0.5">
        <Link
          href="/flow"
          className={cn(
            "flex items-center gap-2.5 px-3 py-2 rounded-md text-sm transition-colors",
            pathname === "/flow"
              ? "bg-secondary text-foreground"
              : "text-muted-foreground hover:text-foreground hover:bg-secondary/50"
          )}
        >
          <span>🏠</span>
          <span>Dashboard</span>
        </Link>

        <div className="pt-2 pb-1">
          <p className="px-3 text-xs font-medium text-muted-foreground uppercase tracking-wider">
            Tools
          </p>
        </div>

        {navItems.map((item) => {
          const isActive =
            pathname.startsWith(item.href)

          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-2.5 px-3 py-2 rounded-md text-sm transition-colors",
                isActive
                  ? "bg-secondary text-foreground font-medium"
                  : "text-muted-foreground hover:text-foreground hover:bg-secondary/50",
                item.highlight && !isActive &&
                  "text-blue-400 hover:text-blue-300"
              )}
            >
              <span>{item.icon}</span>
              <span>{item.label}</span>
              {item.highlight && (
                <span className="ml-auto text-xs bg-blue-600/20 text-blue-400 border border-blue-600/30 px-1.5 py-0.5 rounded-full">
                  AI
                </span>
              )}
            </Link>
          )
        })}
      </nav>

      {/* User section */}
      <div className="p-3 border-t border-border">
        <div className="flex items-center gap-2.5 px-3 py-2">
          <div className="w-7 h-7 rounded-full bg-blue-600/20 border border-blue-600/30 flex items-center justify-center">
            <span className="text-xs font-medium text-blue-400">
              {user?.name?.[0]?.toUpperCase() || "U"}
            </span>
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-xs font-medium truncate">
              {user?.name || "User"}
            </p>
            <p className="text-xs text-muted-foreground truncate">
              {workspace?.plan || "free"}
            </p>
          </div>
          <button
            onClick={logout}
            className="text-xs text-muted-foreground hover:text-foreground transition-colors"
            title="Logout"
          >
            ↪
          </button>
        </div>
      </div>
    </aside>
  )
}

"use client"

import { usePathname } from "next/navigation"

const pageTitles: Record<string, string> = {
  "/dashboard": "Dashboard",
  "/flow": "ZipDesk Flow",
  "/links": "Links",
  "/forms": "Forms",
  "/docs": "Docs",
  "/mail": "Mail",
  "/crm": "CRM",
}

export function Header() {
  const pathname = usePathname()

  const title = Object.entries(pageTitles).find(
    ([path]) => pathname.startsWith(path)
  )?.[1] || "ZipDesk"

  return (
    <header className="h-14 border-b border-border px-6 flex items-center justify-between bg-background">
      <h1 className="text-sm font-medium">
        {title}
      </h1>
      <div className="flex items-center gap-2">
        <span className="text-xs text-muted-foreground bg-secondary px-2 py-1 rounded-full">
          Free plan
        </span>
      </div>
    </header>
  )
}

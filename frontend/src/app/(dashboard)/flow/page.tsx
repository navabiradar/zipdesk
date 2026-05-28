"use client"

import { useState, useRef, useEffect } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { HealthWidget } from "@/components/flow/HealthWidget"
import { EventLog } from "@/components/flow/EventLog"
import { useAuthStore } from "@/stores/authStore"

interface Message {
  id: string
  role: "user" | "assistant"
  content: string
  timestamp: Date
}

const DEMO_PROMPTS = [
  "Create a waitlist form for my startup and send a welcome email to everyone who signs up",
  "Shorten this link: https://example.com/very-long-url",
  "Create a contact form for my website",
  "Show me the system health status",
  "Create a flow: when someone fills my form, add them to my mail list",
]

export default function FlowPage() {
  const { workspace } = useAuthStore()
  const [messages, setMessages] = useState<Message[]>([
    {
      id: "welcome",
      role: "assistant",
      content: `Hi! I'm ZipDesk AI. I can create links, forms, docs, email campaigns, and automations — just by chatting.\n\nTry: "Create a waitlist form and send a welcome email to signups"`,
      timestamp: new Date(),
    },
  ])
  const [input, setInput] = useState("")
  const [streaming, setStreaming] = useState(false)
  const [streamingText, setStreamingText] = useState("")
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({
      behavior: "smooth",
    })
  }, [messages, streamingText])

  async function sendMessage(text?: string) {
    const msg = text || input.trim()
    if (!msg || streaming) return

    setInput("")
    setStreaming(true)
    setStreamingText("")

    // Add user message
    const userMsg: Message = {
      id: Date.now().toString(),
      role: "user",
      content: msg,
      timestamp: new Date(),
    }
    setMessages((prev) => [...prev, userMsg])

    try {
      const token = localStorage.getItem("access_token")
      const API_URL =
        process.env.NEXT_PUBLIC_API_URL ||
        "http://localhost:8080"

      const resp = await fetch(
        `${API_URL}/api/v1/flow/chat`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify({
            message: msg,
            history: messages.slice(-10).map((m) => ({
              role: m.role,
              content: m.content,
            })),
          }),
        }
      )

      if (!resp.ok) {
        throw new Error("Chat request failed")
      }

      const reader = resp.body?.getReader()
      const decoder = new TextDecoder()
      let fullText = ""

      if (reader) {
        while (true) {
          const { done, value } = await reader.read()
          if (done) break

          const chunk = decoder.decode(value)
          const lines = chunk.split("\n")

          for (const line of lines) {
            if (line.startsWith("event: text")) {
              continue
            }
            if (line.startsWith("data: ")) {
              const data = line.slice(6)
              if (data && data !== "done") {
                fullText += data
                setStreamingText(fullText)
              }
            }
            if (line.startsWith("event: done")) {
              break
            }
          }
        }
      }

      // Add assistant message
      const assistantMsg: Message = {
        id: (Date.now() + 1).toString(),
        role: "assistant",
        content: fullText ||
          "I processed your request. Check the event log for details.",
        timestamp: new Date(),
      }
      setMessages((prev) => [...prev, assistantMsg])

    } catch (err) {
      const errorMsg: Message = {
        id: (Date.now() + 1).toString(),
        role: "assistant",
        content:
          "Sorry, I encountered an error. Please check that the backend is running and try again.",
        timestamp: new Date(),
      }
      setMessages((prev) => [...prev, errorMsg])
    } finally {
      setStreaming(false)
      setStreamingText("")
      inputRef.current?.focus()
    }
  }

  function handleKeyDown(
    e: React.KeyboardEvent
  ) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      sendMessage()
    }
  }

  return (
    <div className="h-[calc(100vh-56px)] flex gap-0">

      {/* Chat Panel */}
      <div className="flex-1 flex flex-col min-w-0">

        {/* Messages */}
        <div className="flex-1 overflow-y-auto p-6 space-y-4">

          {messages.map((msg) => (
            <div
              key={msg.id}
              className={`flex gap-3 ${
                msg.role === "user"
                  ? "justify-end"
                  : "justify-start"
              }`}
            >
              {msg.role === "assistant" && (
                <div className="w-7 h-7 rounded-full bg-blue-600 flex items-center justify-center flex-shrink-0 mt-0.5">
                  <span className="text-white text-xs font-bold">
                    Z
                  </span>
                </div>
              )}

              <div
                className={`max-w-[75%] rounded-2xl px-4 py-3 text-sm leading-relaxed whitespace-pre-wrap ${
                  msg.role === "user"
                    ? "bg-blue-600 text-white rounded-br-sm"
                    : "bg-secondary text-foreground rounded-bl-sm"
                }`}
              >
                {msg.content}
              </div>

              {msg.role === "user" && (
                <div className="w-7 h-7 rounded-full bg-secondary flex items-center justify-center flex-shrink-0 mt-0.5">
                  <span className="text-xs font-medium">
                    U
                  </span>
                </div>
              )}
            </div>
          ))}

          {/* Streaming message */}
          {streaming && streamingText && (
            <div className="flex gap-3 justify-start">
              <div className="w-7 h-7 rounded-full bg-blue-600 flex items-center justify-center flex-shrink-0 mt-0.5">
                <span className="text-white text-xs font-bold">
                  Z
                </span>
              </div>
              <div className="max-w-[75%] rounded-2xl rounded-bl-sm px-4 py-3 text-sm bg-secondary text-foreground leading-relaxed whitespace-pre-wrap">
                {streamingText}
                <span className="cursor-blink">▋</span>
              </div>
            </div>
          )}

          {/* Typing indicator */}
          {streaming && !streamingText && (
            <div className="flex gap-3 justify-start">
              <div className="w-7 h-7 rounded-full bg-blue-600 flex items-center justify-center flex-shrink-0">
                <span className="text-white text-xs font-bold">
                  Z
                </span>
              </div>
              <div className="rounded-2xl rounded-bl-sm px-4 py-3 bg-secondary">
                <div className="flex gap-1 items-center h-4">
                  <div className="w-1.5 h-1.5 rounded-full bg-muted-foreground animate-bounce" style={{ animationDelay: "0ms" }} />
                  <div className="w-1.5 h-1.5 rounded-full bg-muted-foreground animate-bounce" style={{ animationDelay: "150ms" }} />
                  <div className="w-1.5 h-1.5 rounded-full bg-muted-foreground animate-bounce" style={{ animationDelay: "300ms" }} />
                </div>
              </div>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>

        {/* Demo prompts */}
        {messages.length <= 1 && (
          <div className="px-6 pb-2">
            <p className="text-xs text-muted-foreground mb-2">
              Try these:
            </p>
            <div className="flex flex-wrap gap-2">
              {DEMO_PROMPTS.slice(0, 3).map(
                (prompt) => (
                  <button
                    key={prompt}
                    onClick={() => sendMessage(prompt)}
                    className="text-xs bg-secondary hover:bg-secondary/80 border border-border rounded-full px-3 py-1.5 text-muted-foreground hover:text-foreground transition-colors text-left max-w-xs truncate"
                  >
                    {prompt}
                  </button>
                )
              )}
            </div>
          </div>
        )}

        {/* Input */}
        <div className="p-4 border-t border-border">
          <div className="flex gap-2 max-w-4xl mx-auto">
            <Input
              ref={inputRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Ask ZipDesk AI to create, analyze or automate anything..."
              disabled={streaming}
              className="flex-1"
              autoFocus
            />
            <Button
              onClick={() => sendMessage()}
              disabled={streaming || !input.trim()}
              size="sm"
              className="px-4"
            >
              {streaming ? "..." : "Send →"}
            </Button>
          </div>
          <p className="text-center text-xs text-muted-foreground mt-2">
            Powered by Claude AI
          </p>
        </div>
      </div>

      {/* Right Panel */}
      <div className="w-72 border-l border-border flex flex-col bg-card">
        <div className="p-4 border-b border-border">
          <h2 className="text-sm font-medium">
            System Status
          </h2>
        </div>
        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          <HealthWidget />
          <EventLog />
        </div>
      </div>

    </div>
  )
}

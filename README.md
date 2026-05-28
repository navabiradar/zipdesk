# ZipDesk

AI-powered suite replacing 6 tools.
Create links, forms, docs, mail &
feedback boards via chat. Auto flows
connect everything. Built in Go.

## Stack
- Backend: Go 1.22 + Fiber v2
- Frontend: Next.js 14 + TypeScript
- Database: PostgreSQL (Supabase)
- Cache: Redis (Upstash)
- AI: Claude API (Anthropic)
- Deploy: Fly.io + Vercel

## Quick Start

### Backend
cd backend
cp .env.example .env
go mod download
make migrate-up
make run

### Frontend
cd frontend
npm install
cp .env.example .env.local
npm run dev

## Architecture
User → AI Chat → Claude API
→ ZipDesk Tools → Event Bus
→ Automations → External Services

## Demo
Type in chat:
"Create a waitlist form and send
 a welcome email to signups"

Watch ZipDesk do it in 15 seconds.

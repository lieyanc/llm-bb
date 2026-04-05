import { HomePage } from "./components/home-page"
import { RoomPage } from "./components/room-page"
import { AdminPage } from "./components/admin-page"
import type { Bootstrap } from "./types"

declare global {
  interface Window {
    __LLM_BB_BOOTSTRAP__?: Bootstrap
  }
}

function getBootstrap() {
  const payload = window.__LLM_BB_BOOTSTRAP__
  if (!payload) {
    throw new Error("missing bootstrap payload")
  }
  return payload
}

export function App() {
  const bootstrap = getBootstrap()

  switch (bootstrap.page) {
    case "home":
      return <HomePage data={bootstrap.data} />
    case "room":
      return <RoomPage data={bootstrap.data} />
    case "admin":
      return <AdminPage data={bootstrap.data} />
    default:
      return null
  }
}

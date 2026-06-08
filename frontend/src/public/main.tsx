import React from "react"
import ReactDOM from "react-dom/client"
import type { PublicBootstrap } from "../shared/types"
import { HomePage } from "./home"
import { RoomPage } from "./room"
import "../shared/index.css"

const stored = localStorage.getItem("theme")
const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches
if (stored === "dark" || (!stored && prefersDark)) {
  document.documentElement.classList.add("dark")
}

function App() {
  const bootstrap = window.__LLM_BB_BOOTSTRAP__ as PublicBootstrap | undefined
  if (!bootstrap) throw new Error("missing bootstrap payload")
  switch (bootstrap.page) {
    case "home":
      return <HomePage data={bootstrap.data} />
    case "room":
      return <RoomPage data={bootstrap.data} />
    default:
      return null
  }
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)

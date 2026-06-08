import React from "react"
import ReactDOM from "react-dom/client"
import type { AdminBootstrap } from "../shared/types"
import { AdminApp } from "./app"
import "../shared/index.css"

const stored = localStorage.getItem("theme")
const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches
if (stored === "dark" || (!stored && prefersDark)) {
  document.documentElement.classList.add("dark")
}

function Root() {
  const bootstrap = window.__LLM_BB_BOOTSTRAP__ as AdminBootstrap | undefined
  if (!bootstrap) throw new Error("missing bootstrap payload")
  return <AdminApp data={bootstrap.data} />
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <Root />
  </React.StrictMode>,
)

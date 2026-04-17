import React from "react"
import ReactDOM from "react-dom/client"
import { App } from "./app"
import "./index.css"

// Apply theme before render to prevent flash
const stored = localStorage.getItem("theme")
const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches
if (stored === "dark" || (!stored && prefersDark)) {
  document.documentElement.classList.add("dark")
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)

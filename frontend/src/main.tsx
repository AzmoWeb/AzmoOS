import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "@fontsource-variable/plus-jakarta-sans";
import "./styles/global.css";
import { App } from "./App";
import { UIProvider } from "./components/overlay";

try {
  const t = localStorage.getItem("azmo-theme");
  if (t) document.documentElement.dataset.theme = t;
} catch { /* stockage indisponible */ }

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <UIProvider>
      <App />
    </UIProvider>
  </StrictMode>,
);

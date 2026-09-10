import React from "react";
import ReactDOM from "react-dom/client";

import App from "@/App";
import { I18nProvider } from "@/i18n";
import { ThemeProvider } from "@/components/ThemeProvider";
import "@/index.css";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <I18nProvider>
      <ThemeProvider>
        <App />
      </ThemeProvider>
    </I18nProvider>
  </React.StrictMode>
);

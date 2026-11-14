import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Toaster } from "sonner";

import App from "./App";
import "./index.css";
import { useAuthStore } from "./state/auth";
import { ThemeProvider } from "./components/ThemeProvider";
import { I18nProvider } from "./i18n";

const queryClient = new QueryClient();
useAuthStore.getState().hydrate();

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <I18nProvider>
        <ThemeProvider>
          <BrowserRouter>
            <App />
          </BrowserRouter>
          <Toaster
            position="top-center"
            duration={3000}
            closeButton
            richColors
            expand={true}
            visibleToasts={3}
            toastOptions={{
              className: "toast-custom",
              style: {
                minWidth: "280px"
              }
            }}
          />
        </ThemeProvider>
      </I18nProvider>
    </QueryClientProvider>
  </React.StrictMode>
);

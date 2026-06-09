import { QueryClient } from "@tanstack/react-query";
import type { AuthTokenResponse } from "../types/api";

const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8081";

if (import.meta.env.DEV) {
  console.log("API URL:", API_URL);
}

// Single in-flight refresh — concurrent 401s share one refresh call.
let refreshPromise: Promise<string | null> | null = null;

async function doRefresh(): Promise<string | null> {
  const refreshToken = localStorage.getItem("guardian_refresh_token");
  if (!refreshToken) return null;
  try {
    const res = await fetch(`${API_URL}/api/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refreshToken }),
    });
    if (!res.ok) return null;
    const data = (await res.json()) as AuthTokenResponse;
    localStorage.setItem("guardian_token", data.token);
    localStorage.setItem("guardian_refresh_token", data.refreshToken);
    localStorage.setItem("guardian_user", JSON.stringify(data.user));
    window.dispatchEvent(new Event("guardian_token_refreshed"));
    return data.token;
  } catch {
    return null;
  }
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const token = localStorage.getItem("guardian_token");
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(init?.headers as Record<string, string> | undefined),
  };

  let res = await fetch(`${API_URL}${path}`, { ...init, headers });

  if (res.status === 401) {
    if (!refreshPromise) {
      refreshPromise = doRefresh().finally(() => {
        refreshPromise = null;
      });
    }
    const newToken = await refreshPromise;

    if (!newToken) {
      localStorage.removeItem("guardian_token");
      localStorage.removeItem("guardian_refresh_token");
      localStorage.removeItem("guardian_user");
      window.location.href = "/login";
      throw new Error("Session expired");
    }

    res = await fetch(`${API_URL}${path}`, {
      ...init,
      headers: { ...headers, Authorization: `Bearer ${newToken}` },
    });
  }

  if (!res.ok) {
    const errorBody = await res.json().catch(() => ({}));
    throw new Error(
      (errorBody as { error?: string }).error || `API error ${res.status}`
    );
  }

  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,
      retry: 1,
    },
  },
});

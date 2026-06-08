export async function fetchJSON<T>(input: RequestInfo | URL, init?: RequestInit): Promise<T> {
  const response = await fetch(input, init)
  if (!response.ok) {
    const payload = (await response.json().catch(() => ({}))) as { error?: string }
    throw new Error(payload.error || "请求失败")
  }
  return (await response.json()) as T
}

export async function postJSON<T>(input: RequestInfo | URL, payload?: unknown, init?: RequestInit): Promise<T> {
  return fetchJSON<T>(input, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers || {}),
    },
    body: payload === undefined ? undefined : JSON.stringify(payload),
    ...init,
  })
}

export async function patchJSON<T>(input: RequestInfo | URL, payload?: unknown, init?: RequestInit): Promise<T> {
  return fetchJSON<T>(input, {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers || {}),
    },
    body: payload === undefined ? undefined : JSON.stringify(payload),
    ...init,
  })
}

export async function deleteJSON<T>(input: RequestInfo | URL, init?: RequestInit): Promise<T> {
  return fetchJSON<T>(input, {
    method: "DELETE",
    ...init,
  })
}

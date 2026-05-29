function setBusy(message) {
  els.libraryStatus.textContent = message;
}

async function api(url, options) {
  const response = await fetch(url, {
    headers: options?.body instanceof FormData ? undefined : { "Content-Type": "application/json" },
    ...options,
  });
  const text = await response.text();
  let payload = {};
  if (text) {
    try {
      payload = JSON.parse(text);
    } catch {
      payload = { error: text };
    }
  }
  if (!response.ok) throw new Error(payload.error || "请求失败");
  return payload;
}


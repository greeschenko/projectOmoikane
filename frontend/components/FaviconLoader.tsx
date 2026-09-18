"use client";

import { useEffect, useState } from "react";

/**
 * Applies the favicon configured in site settings to the document head.
 * Next.js can't render a <link rel="icon"> whose href comes from the CMS at
 * request time, so this client component fetches /api/settings and injects it.
 */
export default function FaviconLoader() {
  const [favicon, setFavicon] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    fetch("/api/settings")
      .then((r) => (r.ok ? r.json() : null))
      .then((data) => {
        if (cancelled) return;
        const value = data?.favicon;
        setFavicon(value && typeof value === "string" ? value : null);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!favicon) return;
    let link = document.head.querySelector<HTMLLinkElement>('link[data-favicon="true"]');
    if (!link) {
      link = document.createElement("link");
      link.rel = "icon";
      link.setAttribute("data-favicon", "true");
      document.head.appendChild(link);
    }
    link.href = favicon;
  }, [favicon]);

  return null;
}
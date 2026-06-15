import { NextResponse } from "next/server";
import store from "@/lib/store";

export async function GET(request: Request) {
  const url = new URL(request.url);
  const path = url.searchParams.get("path") || "";
  const segments = path.split("/").filter(Boolean);
  const page = store.resolvePageByPath(segments);
  if (!page) {
    return NextResponse.json({ error: "Page not found" }, { status: 404 });
  }
  return NextResponse.json(page);
}

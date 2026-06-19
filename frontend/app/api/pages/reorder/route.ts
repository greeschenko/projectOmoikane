import { NextResponse } from "next/server";
import store from "@/lib/store";
import { getSession, requireAdmin } from "@/lib/auth";

export async function PUT(request: Request) {
  const session = await getSession();
  try {
    requireAdmin(session);
  } catch {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }
  const body = await request.json();
  const { parentId, pageIds } = body;
  if (!Array.isArray(pageIds)) {
    return NextResponse.json({ error: "pageIds array required" }, { status: 400 });
  }
  store.reorderPages(parentId ?? null, pageIds);
  return NextResponse.json({ success: true });
}

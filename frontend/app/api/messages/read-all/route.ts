import { NextResponse } from "next/server";
import store from "@/lib/store";
import { getSession, requireAuth } from "@/lib/auth";

export async function POST() {
  const session = await getSession();
  try {
    requireAuth(session);
  } catch {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }
  store.markAllAsRead(session!.userId);
  return NextResponse.json({ success: true });
}

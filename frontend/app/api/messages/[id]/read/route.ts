import { NextResponse } from "next/server";
import store from "@/lib/store";
import { getSession, requireAuth } from "@/lib/auth";

export async function POST(request: Request, { params }: { params: Promise<{ id: string }> }) {
  const session = await getSession();
  try {
    requireAuth(session);
  } catch {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }
  const { id } = await params;
  const msg = store.markAsRead(id, session!.userId);
  if (!msg) {
    return NextResponse.json({ error: "Message not found" }, { status: 404 });
  }
  return NextResponse.json({ success: true });
}

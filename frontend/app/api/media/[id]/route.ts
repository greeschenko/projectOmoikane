import { NextResponse } from "next/server";
import { getSession, requireAdmin } from "@/lib/auth";
import store from "@/lib/store";

export async function DELETE(
  _request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const session = await getSession();
    requireAdmin(session);
  } catch {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const { id } = await params;
  const deleted = store.deleteMedia(id);
  if (!deleted) {
    return NextResponse.json({ error: "Media not found" }, { status: 404 });
  }
  return NextResponse.json({ success: true });
}

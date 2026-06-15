import { NextResponse } from "next/server";
import store from "@/lib/store";
import { getSession, requireAdmin } from "@/lib/auth";

export async function PUT(request: Request, { params }: { params: Promise<{ id: string }> }) {
  const session = await getSession();
  try {
    requireAdmin(session);
  } catch {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }
  const { id } = await params;
  const body = await request.json();
  const page = store.updatePage(id, body);
  if (!page) {
    return NextResponse.json({ error: "Page not found" }, { status: 404 });
  }
  return NextResponse.json(page);
}

export async function DELETE(request: Request, { params }: { params: Promise<{ id: string }> }) {
  const session = await getSession();
  try {
    requireAdmin(session);
  } catch {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }
  const { id } = await params;
  if (!store.deletePage(id)) {
    return NextResponse.json({ error: "Page not found" }, { status: 404 });
  }
  return NextResponse.json({ success: true });
}

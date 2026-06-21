import { NextResponse } from "next/server";
import store from "@/lib/store";
import { getSession } from "@/lib/auth";

export const dynamic = "force-dynamic";

export async function PUT(
  request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  const session = await getSession();
  if (!session) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });

  const { id } = await params;
  const body = await request.json();
  const tag = store.updateTag(id, body);
  if (!tag) return NextResponse.json({ error: "Not Found" }, { status: 404 });
  return NextResponse.json(tag);
}

export async function DELETE(
  _request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  const session = await getSession();
  if (!session) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });

  const { id } = await params;
  const deleted = store.deleteTag(id);
  if (!deleted) return NextResponse.json({ error: "Not Found" }, { status: 404 });
  return NextResponse.json({ success: true });
}

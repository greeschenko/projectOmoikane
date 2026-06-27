import { NextResponse } from "next/server";
import store from "@/lib/store";
import { getSession } from "@/lib/auth";

export const dynamic = "force-dynamic";

export async function GET(
  _request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  const { id } = await params;
  const post = store.getBlogPost(id);
  if (!post) return NextResponse.json({ error: "Not Found" }, { status: 404 });
  return NextResponse.json(post);
}

export async function PUT(
  request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  const session = await getSession();
  if (!session) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });

  const { id } = await params;
  const existing = store.getBlogPost(id);
  if (!existing) return NextResponse.json({ error: "Not Found" }, { status: 404 });
  if (existing.authorId !== session.userId && session.role !== "admin") {
    return NextResponse.json({ error: "Forbidden" }, { status: 403 });
  }

  const body = await request.json();
  const post = store.updateBlogPost(id, body);
  return NextResponse.json(post);
}

export async function DELETE(
  _request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  const session = await getSession();
  if (!session) return NextResponse.json({ error: "Unauthorized" }, { status: 401 });

  const { id } = await params;
  const existing = store.getBlogPost(id);
  if (!existing) return NextResponse.json({ error: "Not Found" }, { status: 404 });
  if (existing.authorId !== session.userId && session.role !== "admin") {
    return NextResponse.json({ error: "Forbidden" }, { status: 403 });
  }

  const deleted = store.deleteBlogPost(id);
  if (!deleted) return NextResponse.json({ error: "Not Found" }, { status: 404 });
  return NextResponse.json({ success: true });
}

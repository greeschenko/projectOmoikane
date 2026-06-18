import { NextResponse } from "next/server";
import store from "@/lib/store";
import { getSession, requireAuth } from "@/lib/auth";

export async function GET() {
  const session = await getSession();
  try {
    requireAuth(session);
  } catch {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }
  const messages = store.getMessages();
  return NextResponse.json({ messages, unreadCount: store.getUnreadCount(session!.userId) });
}

export async function POST(request: Request) {
  const session = await getSession();
  if (!session || session.role !== "admin") {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }
  const body = await request.json();
  const { title, content } = body;
  if (!title) {
    return NextResponse.json({ error: "Title is required" }, { status: 400 });
  }
  const message = store.createMessage({ title, content: content || "" });
  return NextResponse.json(message, { status: 201 });
}

export async function DELETE() {
  const session = await getSession();
  if (!session || session.role !== "admin") {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }
  store.clearMessages();
  return NextResponse.json({ success: true });
}

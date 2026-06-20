import { NextResponse } from "next/server";
import store from "@/lib/store";
import { getSession } from "@/lib/auth";

export async function GET() {
  const session = await getSession();
  if (!session) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }
  const user = store.getUser(session.userId);
  if (!user) {
    return NextResponse.json({ error: "User not found" }, { status: 404 });
  }
  return NextResponse.json({ name: user.name, email: user.email, avatar: user.avatar });
}

export async function PUT(request: Request) {
  const session = await getSession();
  if (!session) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const body = await request.json();
  const { name, email, avatar } = body;

  if (name !== undefined && !name.trim()) {
    return NextResponse.json({ error: "Name cannot be empty" }, { status: 400 });
  }

  const user = store.getUser(session.userId);
  if (!user) {
    return NextResponse.json({ error: "User not found" }, { status: 404 });
  }

  const updates: Record<string, string> = {};
  if (name !== undefined) updates.name = name;
  if (email !== undefined) updates.email = email;
  if (avatar !== undefined) updates.avatar = avatar;

  const updated = store.updateUser(session.userId, updates as any);
  if (!updated) {
    return NextResponse.json({ error: "Failed to update profile" }, { status: 500 });
  }
  return NextResponse.json(updated);
}

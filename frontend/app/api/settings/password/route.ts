import { NextResponse } from "next/server";
import store, { verifyPassword } from "@/lib/store";
import { getSession } from "@/lib/auth";

export async function POST(request: Request) {
  const session = await getSession();
  if (!session) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  const body = await request.json();
  const { currentPassword, newPassword } = body;

  if (!currentPassword || !newPassword) {
    return NextResponse.json({ error: "Current and new password required" }, { status: 400 });
  }

  if (newPassword.length < 6) {
    return NextResponse.json({ error: "Password must be at least 6 characters" }, { status: 400 });
  }

  const user = store.getUser(session.userId);
  if (!user) {
    return NextResponse.json({ error: "User not found" }, { status: 404 });
  }

  if (!verifyPassword(currentPassword, user.password)) {
    return NextResponse.json({ error: "Current password is incorrect" }, { status: 400 });
  }

  store.updateUser(session.userId, { password: newPassword });
  return NextResponse.json({ success: true });
}

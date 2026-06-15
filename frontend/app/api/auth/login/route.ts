import { NextResponse } from "next/server";
import store from "@/lib/store";
import { verifyPassword } from "@/lib/store";
import { setSession } from "@/lib/auth";

export async function POST(request: Request) {
  const body = await request.json();
  const { email, password } = body;
  if (!email || !password) {
    return NextResponse.json({ error: "Email and password required" }, { status: 400 });
  }
  const user = store.getUserByEmail(email);
  if (!user || !verifyPassword(password, user.password)) {
    return NextResponse.json({ error: "Invalid credentials" }, { status: 401 });
  }
  await setSession(user.id, user.role);
  return NextResponse.json({ success: true, role: user.role });
}

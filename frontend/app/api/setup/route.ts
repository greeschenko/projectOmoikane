import { NextResponse } from "next/server";
import store from "@/lib/store";

export async function POST(request: Request) {
  if (store.userCount > 0) {
    return NextResponse.json({ error: "Setup already completed" }, { status: 400 });
  }
  const body = await request.json();
  const { email, password } = body;
  if (!email || !password) {
    return NextResponse.json({ error: "Email and password required" }, { status: 400 });
  }
  store.createUser({ name: "Admin", email, password, role: "admin" });
  return NextResponse.json({ success: true });
}

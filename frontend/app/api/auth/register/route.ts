import { NextResponse } from "next/server";
import store from "@/lib/store";

export async function POST(request: Request) {
  const body = await request.json();
  const { name, email, password } = body;
  if (!name || !email || !password) {
    return NextResponse.json({ error: "All fields required" }, { status: 400 });
  }
  if (store.getUserByEmail(email)) {
    return NextResponse.json({ error: "Email already registered" }, { status: 400 });
  }
  const user = store.createUser({ name, email, password, role: "user" });
  return NextResponse.json({ success: true, user });
}

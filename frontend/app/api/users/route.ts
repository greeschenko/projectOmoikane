import { NextResponse } from "next/server";
import store from "@/lib/store";
import { getSession, requireAdmin } from "@/lib/auth";

export async function GET() {
  const session = await getSession();
  try {
    requireAdmin(session);
  } catch {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }
  return NextResponse.json(store.getUsers());
}

export async function POST(request: Request) {
  const session = await getSession();
  try {
    requireAdmin(session);
  } catch {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }
  const body = await request.json();
  const { name, email, password, role } = body;
  if (!name || !email || !password) {
    return NextResponse.json({ error: "All fields required" }, { status: 400 });
  }
  const user = store.createUser({
    name,
    email,
    password,
    role: role || "user",
  });
  return NextResponse.json(user, { status: 201 });
}
